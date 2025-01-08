package main

import (
	"context"
	"encoding/json"
	"fmt"
	pb "grpcd/canf22g2/grpc"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type MsgGood struct {
	FinalGood string `json:"success"`
}

type MsgBad struct {
	FinalBad string `json:"bad"`
}

const (
	fotaImage = "/tmp/update.bin"
	//u-boot settings
	//ubootBinary    = "/tmp/fota/u-boot.bin"
	//ubootPartition = "/dev/mmcblk0p3"

	//env settings
	ubootEnvBinary    = "/tmp/fota/u-boot_env.bin"
	ubootEnvPartition = "/dev/mtd1"
	//dtb
	dtbBinary    = "/tmp/fota/leipzig.dtb"
	dtbPartition = "/dev/mtd2"
	//Kernel
	kernelBinary    = "/tmp/fota/Image"
	kernelPartition = "/dev/mtd3"
	//rootfs
	rootfsBinary    = "/tmp/fota/rootfs.squashfs"
	rootfsPartition = "/dev/mtd4"
	//daemon
	daemonBinary = "/tmp/fota/daemon.tar"
	daemonPath   = "/mnt/getac"
	//flash
	flashBinary = "/tmp/fota/flash.tar"
	flashPath   = "/mnt/flash"

	tmpFotaFolder = "/tmp/fota"
	cmdNandWrite  = "nandwrite"
	cmdNandErase  = "flash_erase"
)

type FotaServer struct {
	pb.UnimplementedFotaServiceServer
	mu sync.Mutex // To protect shared state across clients
}

var clientID string

/*
finalGood is bit value to indicate which region are FOTA

	LSB            0        0        0        0        0        0        0        0
	Partition    uboot  ubootenv    dtb     Kernel   Rootfs   Getac    Flash  revsersed
*/
func (s *FotaServer) Fota(stream pb.FotaService_FotaServer) error {
	//s.mu.Lock()
	//defer s.mu.Unlock()
	var file *os.File
	var fileSize int64
	var finalGood int
	var finalBad int

	serverID := AppConfig.System.SerialNo

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			Log.Infof("Starting OTA update...")

			for i := 1; i <= 100; i += 10 {
				time.Sleep(300 * time.Millisecond)
				progress := &pb.FotaStatus{
					ClientId: serverID,
					Success:  true,
					Message:  fmt.Sprintf("RESP:OTA Update Progress:%d%%", i),
				}
				if err := stream.Send(progress); err != nil {
					Log.Infof("Error sending progress to client %s:%v", serverID, err)
					return err
				}
			}

			Log.Infof("Upload completed for client %s", serverID)
			progress := &pb.FotaStatus{
				ClientId: serverID,
				Success:  true,
				Message:  "RESP:Upload Image Done",
			}
			if err := stream.Send(progress); err != nil {
				Log.Infof("Error sending progress to client %s: %v", serverID, err)
				return err
			}
			break
		}

		if err != nil {
			if err == context.Canceled {
				Log.Infof("Client %s canceled the connection", clientID)
			} else {
				Log.Infof("Error receiving chunk from client %s: %v", clientID, err)
			}
			return err
		}

		clientID = chunk.GetClientId()
		if file == nil {
			//originalFilename := chunk.GetFilename()
			// Append clientID to filename to avoid conflicts
			//filename = fmt.Sprintf("%s_%s", clientID, originalFilename)

			file, err = os.Create(fotaImage)
			if err != nil {
				Log.Infof("Failed to create file for client %s: %v", clientID, err)
				return fmt.Errorf("failed to create file: %v", err)
			}
			defer file.Close()
		}

		n, err := file.Write(chunk.GetContent())
		if err != nil {
			Log.Infof("Failed to write chunk for client %s: %v", clientID, err)
			return fmt.Errorf("failed to write chunk: %v", err)
		}

		fileSize += int64(n)
		//log.Printf("Client %s: received chunk of size %d, total size %d", clientID, n, fileSize)
	}

	Log.Warnf("who is host:%s", clientID)
	finalGood = 0x00
	finalBad = 0x00

	Log.Info("========== Run Extract file ==========")
	if err := FotaExtractFile(); err != nil {
		stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
		finalBad = 0xFF
	} else {
		stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Extract file success"})

		Log.Info("========== Run FotaUbootEnv ==========")
		if err := FotaUbootEnv(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Skip UbootEnv"})
				finalGood = finalGood & (^(1 << 1))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<1
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash UbootEnv success"})
			finalGood = finalGood | (1 << 1)

		}
		Log.Info("========== Run FotaDTB ==========")
		if err := FotaDtb(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: "RESP:Skip DTB"})
				finalGood = finalGood & (^(1 << 2))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<2
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash DTB success"})
			finalGood = finalGood | (1 << 2)
		}
		Log.Info("========== Run FotaKernel ==========")
		if err := FotaKernel(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: "RESP:Skip Kernal"})
				finalGood = finalGood & (^(1 << 3))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<3
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash Kernel success"})
			finalGood = finalGood | (1 << 3)
		}
		Log.Info("========== Run FotaRootFS ==========")
		if err := FotaRootFs(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: "RESP:Skip RootFS"})
				finalGood = finalGood & (^(1 << 4))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<4
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash RootFS success"})
			finalGood = finalGood | (1 << 4)
		}
		Log.Info("========== Run FotaGetacDaemon ==========")
		if err := FotaDaemon(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: "RESP:Skip Getac Region"})
				finalGood = finalGood & (^(1 << 5))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<5
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash Region Getac success"})
			finalGood = finalGood | (1 << 5)
		}
		Log.Info("========== Run FotaFlashDaemon ==========")
		if err := FotaFlash(); err != nil {
			if err.Error() == "file not found" {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: "RESP:Skip Flash Region"})
				finalGood = finalGood & (^(1 << 6))
			} else {
				stream.Send(&pb.FotaStatus{ClientId: serverID, Success: false, Message: err.Error()})
				finalBad = finalBad | 1<<6
			}
		} else {
			stream.Send(&pb.FotaStatus{ClientId: serverID, Success: true, Message: "RESP:Flash Region Flash success"})
			finalGood = finalGood | (1 << 6)
		}
	}
	Log.Info("Final Report")
	if finalBad != 0x00 {
		Log.Infof("FinalBad:0x%x", finalBad)
		msg := MsgBad{
			FinalBad: strconv.FormatInt(int64(finalBad), 16),
		}
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			Log.Infof("Error marshalling JSON:", err)
		}
		finalMessage := fmt.Sprintf("RESP:%s", string(jsonMsg))
		stream.Send(&pb.FotaStatus{
			ClientId: "serverID",
			Success:  true,
			Message:  finalMessage,
		})

	} else {
		Log.Infof("FinalGood:0x%x", finalGood)
		msg := MsgGood{
			FinalGood: strconv.FormatInt(int64(finalGood), 16),
		}
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			Log.Infof("Error marshalling JSON:", err)
		}
		finalMessage := fmt.Sprintf("RESP:%s", string(jsonMsg))

		fmt.Println("JSON Message:", finalMessage)
		stream.Send(&pb.FotaStatus{
			ClientId: "serverID",
			Success:  false,
			Message:  finalMessage,
		})
	}

	go func() {
		time.Sleep(5 * time.Second)
		if len(clientID) > 0 && clientID != "DVR" { //For DVR, it will reset POE directly
			Log.Warnf("Fota done, System rebooting")
			cmd := exec.Command("/bin/sh", "-c", "echo 0 > /sys/devices/platform/secure-monitor/boot_flag")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				Log.Infof("error to set boot flag: %v", err)
			}
			cmd = exec.Command("/bin/sh", "-c", "reboot")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				Log.Infof("error to reboot: %v", err)
			}
		} else {
			Log.Warnf("Fota done, No Reboot, Quit")
		}
	}()

	return nil
}

func PrehookInstall() error {
	cmd := exec.Command("./web/sh/prehookInstall.sh")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Log.Infof("Fail to run PrehookInstall: %v", err)
		return fmt.Errorf("error running PrehookInstall:%w", err)
	}
	return nil
}

func FotaExtractFile() error {
	if _, err := os.Stat(fotaImage); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", fotaImage)
		return fmt.Errorf("Error:Image not found")
	}
	Log.Infof("File %s exists\n", fotaImage)

	if _, err := os.Stat(tmpFotaFolder); os.IsNotExist(err) {
		err := os.Mkdir(tmpFotaFolder, 0755)
		if err != nil {
			Log.Infof("Error creating directory %s: %v", tmpFotaFolder, err)
			return fmt.Errorf("Error:creating directory %s: %w", tmpFotaFolder, err)
		}
	}

	// Extract the fota.img to /tmp/fota folder
	cmd := exec.Command("tar", "--strip-components=1", "-xf", fotaImage, "-C", tmpFotaFolder)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Log.Infof("Error extracting %s: %v", fotaImage, err)
		return fmt.Errorf("Error:extracting file %s: %w", fotaImage, err)
	}
	return nil
}

func FotaUbootEnv() error {
	if _, err := os.Stat(ubootEnvBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", ubootEnvBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", ubootEnvBinary)
	}

	cmd1 := exec.Command(cmdNandErase, ubootEnvPartition, "0", "0")
	cmd1.Env = os.Environ()
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err := cmd1.Run()
	if err != nil {
		Log.Infof("Error erasing command: %v", err)
		return fmt.Errorf("error erasing command: %w", err)
	}

	cmd2 := exec.Command(cmdNandWrite, "-p", ubootEnvPartition, ubootEnvBinary)
	cmd2.Env = os.Environ()
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err = cmd2.Run()
	if err != nil {
		Log.Infof("Error flashing command: %v", err)
		return fmt.Errorf("error flashing command: %w", err)
	}
	return nil
}

func FotaKernel() error {
	if _, err := os.Stat(kernelBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", kernelBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", kernelBinary)
	}

	cmd1 := exec.Command(cmdNandErase, kernelPartition, "0", "0")
	cmd1.Env = os.Environ()
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err := cmd1.Run()
	if err != nil {
		Log.Infof("Error erasing command: %v", err)
		return fmt.Errorf("error erasing command: %w", err)
	}

	cmd2 := exec.Command(cmdNandWrite, "-p", kernelPartition, kernelBinary)
	cmd2.Env = os.Environ()
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err = cmd2.Run()
	if err != nil {
		Log.Infof("Error erasing command: %v", err)
		return fmt.Errorf("error erasing command: %w", err)
	}
	return nil
}

func FotaDtb() error {
	if _, err := os.Stat(dtbBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", dtbBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", dtbBinary)
	}

	cmd1 := exec.Command(cmdNandErase, dtbPartition, "0", "0")
	cmd1.Env = os.Environ()
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err := cmd1.Run()
	if err != nil {
		Log.Infof("Error erasing command: %v", err)
		return fmt.Errorf("error erasing command: %w", err)
	}

	cmd2 := exec.Command(cmdNandWrite, "-p", dtbPartition, dtbBinary)
	cmd2.Env = os.Environ()
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err = cmd2.Run()
	if err != nil {
		Log.Infof("Error flashing command: %v", err)
		return fmt.Errorf("error flashing command: %w", err)
	}

	return nil
}

func FotaRootFs() error {
	if _, err := os.Stat(rootfsBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", rootfsBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", rootfsBinary)
	}

	cmd1 := exec.Command(cmdNandErase, rootfsPartition, "0", "0")
	cmd1.Env = os.Environ()
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err := cmd1.Run()
	if err != nil {
		Log.Infof("Error erasing command: %v", err)
		return fmt.Errorf("error erasing command: %w", err)
	}

	cmd2 := exec.Command(cmdNandWrite, "-p", rootfsPartition, rootfsBinary)
	cmd2.Env = os.Environ()
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err = cmd2.Run()
	if err != nil {
		Log.Infof("Error flashing command: %v", err)
		return fmt.Errorf("error flashing command: %w", err)
	}
	return nil
}

func FotaDaemon() error {
	cmd := exec.Command("/mnt/getac/bin/sh/cleanup.sh", "getac")
	err := cmd.Run()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err != nil {
		Log.Infof("Error cleanup.sh: %v", err)
	}

	if _, err := os.Stat(daemonBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", daemonBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", daemonBinary)
	}

	extractDirPath := "/tmp/tmp_daemon"
	localPath := "/mnt/getac"

	err = os.MkdirAll(extractDirPath, 0755)
	if err != nil {
		Log.Infof("Error creating directory: %v", err)
	} else {
		Log.Infof("Directory created successfully: %s", extractDirPath)
	}

	if _, err := os.Stat(extractDirPath); os.IsNotExist(err) {
		Log.Infof("Directory does not exist: %s", extractDirPath)
	} else {
		Log.Infof("Directory exists: %s", extractDirPath)
	}

	//Get Folder list
	cmd = exec.Command("tar", "--strip-components=1", "-xf", daemonBinary, "-C", extractDirPath)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		Log.Infof("Error tar command: %v", err)
	}

	//Get the list of files/folders extracted
	contents, err := os.ReadDir(extractDirPath)
	if err != nil {
		Log.Infof("Error reading directory contents: %v", err)
	}
	Log.Infof("%v\n", contents)

	for _, item := range contents {
		itemPath := filepath.Join(localPath, item.Name())
		if item.IsDir() {
			// If it's a directory, remove it
			err = os.RemoveAll(itemPath)
			if err != nil {
				Log.Infof("Error removing directory: %v", err)
			} else {
				Log.Infof("Removed directory: %s", itemPath)
			}
		} else {
			// If it's a file, remove it
			err = os.Remove(itemPath)
			if err != nil {
				Log.Infof("Error removing file: %v", err)
			} else {
				Log.Infof("Removed file: %s", itemPath)
			}
		}
	}

	cmd = exec.Command("tar", "--strip-components=1", "-xf", daemonBinary, "-C", daemonPath)
	err = cmd.Run()
	if err != nil {
		Log.Infof("Error : %v", err)
		return fmt.Errorf("error tar: %w", err)
	}

	cmd = exec.Command("sync")
	err = cmd.Run()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err != nil {
		Log.Infof("Error sync: %v", err)
	}
	return nil
}

func FotaFlash() error {
	if _, err := os.Stat(flashBinary); os.IsNotExist(err) {
		Log.Infof("File %s does not exist.\n", flashBinary)
		return fmt.Errorf("file not found")
	} else {
		Log.Infof("File %s exists.\n", flashBinary)
	}

	extractDirPath := "/tmp/tmp_flash"
	localPath := "/mnt/flash"

	err := os.MkdirAll(extractDirPath, 0755)
	if err != nil {
		Log.Infof("Error creating directory: %v", err)
	} else {
		Log.Infof("Directory created successfully: %s", extractDirPath)
	}

	if _, err := os.Stat(extractDirPath); os.IsNotExist(err) {
		Log.Infof("Directory does not exist: %s", extractDirPath)
	} else {
		Log.Infof("Directory exists: %s", extractDirPath)
	}

	//Get Folder list
	cmd := exec.Command("tar", "--strip-components=1", "-xf", flashBinary, "-C", extractDirPath)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		Log.Infof("Error tar command: %v", err)
	}

	//Get the list of files/folders extracted
	contents, err := os.ReadDir(extractDirPath)
	if err != nil {
		Log.Infof("Error reading directory contents: %v", err)
	}
	Log.Infof("%v\n", contents)

	//Remove files or directories based on their type
	for _, item := range contents {
		itemPath := filepath.Join(localPath, item.Name())
		if item.IsDir() {
			// If it's a directory, remove it
			err = os.RemoveAll(itemPath)
			if err != nil {
				Log.Infof("Error removing directory: %v", err)
			} else {
				Log.Infof("Removed directory: %s", itemPath)
			}
		} else {
			// If it's a file, remove it
			err = os.Remove(itemPath)
			if err != nil {
				Log.Infof("Error removing file: %v", err)
			} else {
				Log.Infof("Removed file: %s", itemPath)
			}
		}
	}

	cmd = exec.Command("tar", "--strip-components=1", "-xf", flashBinary, "-C", flashPath)
	err = cmd.Run()
	if err != nil {
		Log.Infof("Error extracting command: %v", err)
		return fmt.Errorf("error extracting command: %w", err)
	}

	cmd = exec.Command("/mnt/getac/bin/sh/cleanup.sh", "flash")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		Log.Infof("Error running cleanup.sh: %v", err)
	}

	cmd = exec.Command("sync")
	err = cmd.Run()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err != nil {
		Log.Infof("sync: %v", err)
	}
	return nil
}
