package main

import (
	"context"
	"flag"
	"io"
	"os"
	"runtime"
	"time"

	pb "grpcd/canf22g2/grpc"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"
)

var clientID string
var Log = logrus.New()

func main() {
	flag.StringVar(&clientID, "id", "kc", "Input your ID")
	flag.Parse()
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		Log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFotaServiceClient(conn)

	// Open the file to be uploaded
	file, err := os.Open("update.bin")
	if err != nil {
		Log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create a gRPC stream
	stream, err := client.Fota(context.Background())
	if err != nil {
		Log.Infof("failed to create stream: %v", err)
	}

	// Buffer to read file chunks
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			// Finished reading file
			break
		}
		if err != nil {
			Log.Infof("failed to read file: %v", err)
		}

		// Send file chunk to the server
		err = stream.Send(&pb.FileChunk{
			ClientId: clientID,
			Filename: "update.bin",
			Content:  buf[:n],
		})
		if err != nil {
			Log.Infof("failed to send file chunk: %v", err)
		}
	}

	// Close the stream after sending all file chunks
	go func() {
		time.Sleep(2 * time.Second) // Simulating delay
		stream.CloseSend()
	}()

	// Receive OTA update progress from the server
	for {
		status, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			Log.Infof("failed to receive status: %v", err)
		}

		Log.Infof("OTA Status: %s\n", status.Message)
	}
}

func init() {
	Log.SetLevel(logrus.DebugLevel)
	Log.SetReportCaller(true)
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:           "2006-01-02 15:04:05",
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		DisableLevelTruncation:    true,
	})
	Log.SetOutput(io.MultiWriter(
		os.Stdout,
		&lumberjack.Logger{
			Filename:   "/mnt/flash/logger_storage/APLog/grpcd.log",
			MaxSize:    1, // megabytes
			MaxBackups: 5,
			MaxAge:     1,     //days
			Compress:   false, // disabled by default
		}))
	numProcs := runtime.GOMAXPROCS(0)
	runtime.GOMAXPROCS(numProcs)
	Log.Infoln("GOMAXPROCS set to:", numProcs)
}
