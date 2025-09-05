package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yaninyzwitty/grpc-device-logging/config"
	"github.com/yaninyzwitty/grpc-device-logging/db"
	"github.com/yaninyzwitty/grpc-device-logging/device"
	devicev1 "github.com/yaninyzwitty/grpc-device-logging/gen/device/v1"
	mon "github.com/yaninyzwitty/grpc-device-logging/metrics"
	"github.com/yaninyzwitty/grpc-device-logging/util"
	"google.golang.org/grpc"
)

type server struct {
	db  *pgxpool.Pool
	cfg *config.Config
	m   *mon.Metrics
	devicev1.UnimplementedCloudServiceServer
}

func main() {
	cp := flag.String("config", "", "Path to config file")
	flag.Parse()
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := new(config.Config)
	cfg.LoadConfig(*cp)

	if cfg.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	reg := prometheus.NewRegistry()
	mon.StartPrometheusServer(cfg, reg)

	appPort := fmt.Sprintf(":%d", cfg.AppPort)
	lis, err := net.Listen("tcp", appPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	ns := newServer(ctx, cfg, reg)
	devicev1.RegisterCloudServiceServer(s, ns)
	slog.Info("gRPC server starting", "port", cfg.AppPort)

	s.Serve(lis)
}

func newServer(ctx context.Context, cfg *config.Config, reg *prometheus.Registry) *server {
	m := mon.NewMetrics(reg)
	r := server{
		cfg: cfg,
		m:   m,
	}
	r.db = db.DbConnect(ctx, cfg)
	return &r

}

func (s *server) GetDevices(ctx context.Context, req *devicev1.GetDevicesRequest) (*devicev1.GetDevicesResponse, error) {
	ds := []*devicev1.Device{
		{
			Id:        1,
			Uuid:      "9add349c-c35c-4d32-ab0f-53da1ba40a2a",
			Mac:       "EF-2B-C4-F5-D6-34",
			Firmware:  "2.1.5",
			CreatedAt: "2024-05-28T15:21:51.137Z",
			UpdatedAt: "2024-05-28T15:21:51.137Z",
		},
		{
			Id:        2,
			Uuid:      "d2293412-36eb-46e7-9231-af7e9249fffe",
			Mac:       "E7-34-96-33-0C-4C",
			Firmware:  "1.0.3",
			CreatedAt: "2024-01-28T15:20:51.137Z",
			UpdatedAt: "2024-01-28T15:20:51.137Z",
		},
		{
			Id:        3,
			Uuid:      "eee58ca8-ca51-47a5-ab48-163fd0e44b77",
			Mac:       "68-93-9B-B5-33-B9",
			Firmware:  "4.3.1",
			CreatedAt: "2024-08-28T15:18:21.137Z",
			UpdatedAt: "2024-08-28T15:18:21.137Z",
		},
	}

	dr := devicev1.GetDevicesResponse{
		Devices: ds,
	}
	return &dr, nil
}

func (s *server) CreateDevice(ctx context.Context, req *devicev1.CreateDeviceRequest) (*devicev1.CreateDeviceResponse, error) {
	now := time.Now().Format(time.RFC3339Nano)

	// Map request to internal device model
	d := &device.Device{
		Device: devicev1.Device{
			Uuid:      uuid.New().String(),
			Mac:       req.GetMac(),
			Firmware:  req.GetFirmware(),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := d.Insert(ctx, s.db, s.m)
	if err != nil {
		s.m.Errors.With(prometheus.Labels{"op": "insert", "db": "postgres"}).Add(1)
		util.Warn(err, "failed to save device in postgres")
		return nil, err
	}
	slog.Debug("device saved in postgres", "id", d.Device.Id, "mac", d.Device.Mac, "firmware", d.Device.Firmware)

	resp := &devicev1.CreateDeviceResponse{Device: &d.Device}
	return resp, nil

}
