package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	devicev1 "github.com/yaninyzwitty/grpc-device-logging/gen/device/v1"
	mon "github.com/yaninyzwitty/grpc-device-logging/metrics"
	"github.com/yaninyzwitty/grpc-device-logging/util"
)

type Device struct {
	Device devicev1.Device
}

func (d *Device) Insert(ctx context.Context, db *pgxpool.Pool, m *mon.Metrics) (err error) {
	start := time.Now()
	defer func() {
		m.Duration.With(prometheus.Labels{
			"op": "insert",
			"db": "postgres",
		}).Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO grpc_device (uuid, mac, firmware, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err = db.QueryRow(
		ctx,
		query,
		d.Device.Uuid,
		d.Device.Mac,
		d.Device.Firmware,
		d.Device.CreatedAt,
		d.Device.UpdatedAt,
	).Scan(&d.Device.Id)

	return util.Annotate(err, "device insert failed")
}
