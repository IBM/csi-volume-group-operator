package fake

import (
	"time"
	"context"

	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
)

func New(address, driver string) (*grpcClient.Client, error) {
	client := &grpcClient.Client{}
	metricsManager := metrics.NewCSIMetricsManager(driver)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := connection.Connect(ctx, address, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		return client, err
	}
	client.Client = conn
	client.Timeout = time.Minute
	return client, nil
}
