package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yaninyzwitty/grpc-device-logging/config"
	"github.com/yaninyzwitty/grpc-device-logging/db"
	"github.com/yaninyzwitty/grpc-device-logging/device"
	devicev1 "github.com/yaninyzwitty/grpc-device-logging/gen/device/v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	mon "github.com/yaninyzwitty/grpc-device-logging/metrics"
	"github.com/yaninyzwitty/grpc-device-logging/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

type server struct {
	db  *pgxpool.Pool
	cfg *config.Config
	m   *mon.Metrics
	devicev1.UnimplementedCloudServiceServer
}

var (
	system = "" // empty string refers to overall service health
	sleep  = flag.Duration("sleep", time.Second*5, "duration between changes in health")
)

func main() {
	cp := flag.String("config", "", "Path to config file")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(config.Config)
	if err := cfg.LoadConfig(*cp); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

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
	reflection.Register(s)

	// Health check registration
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)

	ns := newServer(ctx, cfg, reg)
	devicev1.RegisterCloudServiceServer(s, ns)

	// Async health toggler
	go func() {
		status := healthpb.HealthCheckResponse_SERVING
		for {
			healthServer.SetServingStatus(system, status)
			if status == healthpb.HealthCheckResponse_SERVING {
				status = healthpb.HealthCheckResponse_NOT_SERVING
			} else {
				status = healthpb.HealthCheckResponse_SERVING
			}
			time.Sleep(*sleep)
		}
	}()

	// Channel to catch OS termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// serve in a go-routine
	go func() {
		if err := s.Serve(lis); err != nil {
			slog.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("gRPC server starting", "port", cfg.AppPort)

	// Block until signal is received
	<-stop

	slog.Warn("Shutdown signal received, marking health as NOT_SERVING...")
	healthServer.SetServingStatus(system, healthpb.HealthCheckResponse_NOT_SERVING)

	// Optional: Drain connections for Envoy/NLB (AWS uses 5s target deregistration delay by default)
	time.Sleep(5 * time.Second)

	gracefulStop := make(chan struct{})

	go func() {
		// Waits for in-flight requests to complete
		s.GracefulStop()
		close(gracefulStop)

	}()

	select {
	case <-gracefulStop:
		slog.Info("Server stopped gracefully ")
	case <-time.After(10 * time.Second): // Grace period timeout
		slog.Warn("Graceful shutdown timed out, forcing stop ")
		s.Stop()

	}

	// Cleanup resources
	slog.Info("Closing database pool...")
	ns.db.Close()

	slog.Info("Shutdown complete ")

}

func newServer(ctx context.Context, cfg *config.Config, reg *prometheus.Registry) *server {
	m := mon.NewMetrics(reg)
	return &server{
		cfg: cfg,
		m:   m,
		db:  db.DbConnect(ctx, cfg),
	}
}

// --- Handler: GetDevices
func (s *server) GetDevices(ctx context.Context, req *devicev1.GetDevicesRequest) (*devicev1.GetDevicesResponse, error) {
	return &devicev1.GetDevicesResponse{
		Devices: []*devicev1.Device{
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
		},
	}, nil
}

// --- Handler: CreateDevice
func (s *server) CreateDevice(ctx context.Context, req *devicev1.CreateDeviceRequest) (*devicev1.CreateDeviceResponse, error) {
	now := time.Now().Format(time.RFC3339Nano)

	d := &device.Device{
		Device: devicev1.Device{
			Uuid:      uuid.New().String(),
			Mac:       req.GetMac(),
			Firmware:  req.GetFirmware(),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if err := d.Insert(ctx, s.db, s.m); err != nil {
		s.m.Errors.With(prometheus.Labels{"op": "insert", "db": "postgres"}).Inc()
		util.Warn(err, "failed to save device in postgres")
		return nil, err
	}

	slog.Debug("device saved in postgres", "id", d.Device.Id, "mac", d.Device.Mac, "firmware", d.Device.Firmware)
	return &devicev1.CreateDeviceResponse{Device: &d.Device}, nil
}
