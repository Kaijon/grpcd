package main

import (
	pb "grpcd/canf22g2/grpc"
	"io"
	"log"
	"net"
	"os"
	"runtime"

	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()

func main() {
	configInit()
	MqttInit()
	go StartMqttInLoop()
	StartMqttWorker()

	go func() {
		logrus.Errorln(http.ListenAndServe(":6060", nil))
	}()
	// It is used on emulator
	//LoadConfigDefault()

	// Start gRPC server
	lis, err := net.Listen("tcp", "[::]:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()

	// register statusLED server
	ledServer := &LEDServer{}
	pb.RegisterLEDServiceServer(srv, ledServer)

	// register DeviceInfo server
	deviceInfoServer := &DeviceInfoServer{}
	pb.RegisterDeviceInfoServiceServer(srv, deviceInfoServer)

	// register NetworkInfo server
	networkInfoServer := &NetworkInfoServer{}
	pb.RegisterNetworkInfoServiceServer(srv, networkInfoServer)

	// register VideoInfo server
	videoInfoServer := &VideoInfoServer{}
	pb.RegisterVideoInfoServiceServer(srv, videoInfoServer)

	// register WatermarkInfo server
	watermarkInfoServer := &WatermarkInfoServer{}
	pb.RegisterWatermarkInfoServiceServer(srv, watermarkInfoServer)

	// register UnifiedFileTransfer server
	unifiedFileTransferServer := &UnifiedFileTransferServer{}
	pb.RegisterUnifiedFileTransferServer(srv, unifiedFileTransferServer)

	// register Lux server
	luxServer := &LuxServer{}
	pb.RegisterLuxServiceServer(srv, luxServer)

	fotaServer := &FotaServer{}
	pb.RegisterFotaServiceServer(srv, fotaServer)

	// Register reflection for debugging
	reflection.Register(srv)
	// start gRPC server
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	Log.SetLevel(logrus.DebugLevel)
	Log.SetReportCaller(true)
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:           "2006-01-02 15:04:05.000000",
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
