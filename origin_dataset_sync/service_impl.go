package origin_dataset_sync

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
)

type serviceImpl struct {
	syncer syncer
}

func (d *serviceImpl) TriggerOriginDatasetsSync(ctx context.Context, _ *connect.Request[dodservice.TriggerOriginDatasetsSyncRequest]) (*connect.Response[dodservice.TriggerOriginDatasetsSyncResponse], error) {
	ctx, span := observability.StartSpan(ctx, "TriggerOriginDatasetsSync")
	defer span.End()

	d.syncer.triggerSync(context.WithoutCancel(ctx))

	return connect.NewResponse(&dodservice.TriggerOriginDatasetsSyncResponse{}), nil
}

func (d *serviceImpl) SyncOriginDataset(ctx context.Context, c *connect.Request[dodservice.SyncOriginDatasetRequest]) (*connect.Response[dodservice.SyncOriginDatasetResponse], error) {
	ctx, span := observability.StartSpan(ctx, "SyncOriginDataset", attribute.String("dataset-accession", c.Msg.GetDatasetAccession()))
	defer span.End()

	if c.Msg.GetDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dataset accession must be provided"))
	}

	if err := d.syncer.syncOriginDataset(ctx, c.Msg.GetDatasetAccession()); err != nil {
		span.Error("failed to sync origin dataset", err)

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&dodservice.SyncOriginDatasetResponse{}), nil
}
