package main

import (
	pb "grpcd/canf22g2/grpc"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"path/filepath"
	"os/exec"
)

const AIPath = "/tmp/"
const AIFile = "alpr_update.tar.gz"
const AIPerform = "/tmp/alpr_update/perform.sh"

type UnifiedFileTransferServer struct {
	pb.UnimplementedUnifiedFileTransferServer
}

func (s *UnifiedFileTransferServer) UploadFirmware(stream pb.UnifiedFileTransfer_UploadFirmwareServer) error {
	Log.Info(">>Run")
	return GeneralUploadFile(stream, firmwareUploadCallback, "")
}

func (s *UnifiedFileTransferServer) UploadAIFile(stream pb.UnifiedFileTransfer_UploadAIFileServer) error {
	AIFilePath := AIPath + AIFile
	return GeneralUploadFile(stream, aiOptionUploadCallback, AIFilePath)
}

// UploadFile adapts to different platforms for file upload.
type UploadCallback func(filename string) error
type StreamReceiver interface {
	Recv() (*pb.UnifiedChunk, error)
}

// GeneralUploadFile: a general file upload function that can be used for different types of file uploads.
func GeneralUploadFile(stream StreamReceiver, callback UploadCallback, path string) error {
	Log.Info(">>Run")
	log.Println("Starting file upload")
	if path == "" {
		if runtime.GOARCH == "amd64" {
			path = "uploaded_file_amd64"
		} else {
			path = "uploaded_file"
		}
	}

	// check the path is exist.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Printf("Error creating directory: %v", err)
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	for {
		retryCount := 3
		var chunk *pb.UnifiedChunk
		for retry := 0; retry < retryCount; retry++ {
			chunk, err = stream.Recv()
			if err == nil || err == io.EOF {
				break
			}
			log.Printf("Error receiving chunk, retrying %d/%d: %v", retry+1, retryCount, err)
		}
		if err == io.EOF {
			log.Println("File upload completed")
			break
		}
		if err != nil {
			log.Printf("Error receiving chunk after retries: %v", err)
			return err
		}

		_, writeErr := file.Write(chunk.Content)
		if writeErr != nil {
			log.Printf("Error writing chunk to file: %v", writeErr)
			return writeErr
		}
	}

	log.Printf("Invoking callback for file: %s", path)
	return callback(path)
}

func renameFile(oldName, newName string) error {
	Log.Info(">>Run")
	if err := os.Rename(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %v", oldName, newName, err)
	}
	return nil
}

func firmwareUploadCallback(filename string) error {
	Log.Info(">>Run")
	fmt.Println("Firmware file uploaded:", filename)
	// change the uploaded_file_amd64 file name to fwUpdate.file if the architecture is amd64
	if runtime.GOARCH == "amd64" {
		return renameFile("uploaded_file_amd64", "fwUpdate.file")
	}
	// implement the firmware bin update logic here

	return nil
}

func aiOptionUploadCallback(filename string) error {
	Log.Info(">>Run")
	fmt.Println("AI Option file uploaded:", filename)
	// implement the AI option bin update logic here

	cmd := exec.Command("sh", "-c", "openssl des3 -d -k T7%f3aG1@xLq -salt -in "+AIFile+" | gunzip | tar xf -")
	cmd.Dir = AIPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Command execution failed: %v\nOutput: %s", err, output)
	}
	log.Printf("Command output:\n%s", output)

	cmd = exec.Command("sh", AIPerform)
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error executing script %s: %v\nOutput: %s", AIPerform, err, output)
	}
	log.Printf("Script executed successfully:\n%s", output)

	return nil
}
