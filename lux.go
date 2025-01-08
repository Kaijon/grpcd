package main

import (
	"context"
	pb "grpcd/canf22g2/grpc"
)

type LuxServer struct {
	pb.UnimplementedLuxServiceServer
}

func (s *LuxServer) GetDayNightMode(ctx context.Context, req *pb.GetDayNightModeRequest) (*pb.GetDayNightModeResponse, error) {
	Log.Info(">>Run")
	dayNightMode := AppConfig.DayNightMode.Mode
	return &pb.GetDayNightModeResponse{DayNightMode: dayNightMode}, nil
}
