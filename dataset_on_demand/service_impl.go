package dataset_on_demand

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/public/v1"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/on_demand_dataset_metadata_file_handler"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"go.opentelemetry.io/otel/attribute"
)

type serviceImpl struct{}

func (d *serviceImpl) RequestDatasetCreation(ctx context.Context, c *connect.Request[dodservice.RequestDatasetCreationRequest]) (*connect.Response[dodservice.RequestDatasetCreationResponse], error) {
	token, ok := authn.GetInfo(ctx).(jwt.Token)
	if !ok || token.Subject() == "" {
		slog.Error("could not find authn value in context")

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	user := token.Subject()
	ctx, span := observability.StartSpan(ctx, "RequestDatasetCreation", attribute.String("user", user))
	defer span.End()

	if len(c.Msg.GetImageAccessions()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no image accessions requested"))
	}

	// calculate the hash of the requested image accession to check for already existing on demand dataset with the same set of images
	imageAccessionHash := hashImageAccessions(c.Msg.GetImageAccessions())
	// Possible future improvement, lock by the imageAccessionHash to avoid race condition when multiple concurrent requests to create same dataset

	existingOnDemandDatasetAccession, err := database.GetOnDemandDatasetAccessionFromImageAccessionsHash(ctx, imageAccessionHash)
	if err != nil {
		span.Error("failed to check for existing on demand dataset image accessions hash", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if existingOnDemandDatasetAccession != "" {
		return connect.NewResponse(&dodservice.RequestDatasetCreationResponse{
			OnDemandDatasetAccession: existingOnDemandDatasetAccession,
		}), nil
	}

	originDatasetImages := make(map[string]map[string]struct{})

	// Create a new context without cancel such that if user cancels the request we still proceed to finish the on demand dataset registration
	// This is to avoid scenarios where caller cancels the request while uploading and triggering ingestion for the metadata files, and where it could end up in a partial state
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		span.Error("failed to begin database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			span.Error("failed to rollback database transaction", err)
		}
	}()

	for _, imageAccession := range c.Msg.GetImageAccessions() {
		originDatasetAccession, err := tx.GetOriginDatasetAccessionFromImageAccession(ctx, imageAccession)
		if err != nil {
			span.Error("failed to get origin dataset accession from image accession", err, slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDatasetAccession == "" {
			span.Warn("no dataset found from image accession", slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		if _, ok := originDatasetImages[originDatasetAccession]; !ok {
			originDatasetImages[originDatasetAccession] = map[string]struct{}{imageAccession: {}}

			continue
		}

		if _, ok := originDatasetImages[originDatasetAccession][imageAccession]; ok {
			span.Info("user requested to combine identical image accessions", slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duplicate image accession requested"))
		}

		originDatasetImages[originDatasetAccession][imageAccession] = struct{}{}
	}

	originDatasets := make(map[string]*models.OriginDataset)

	var ensureSameWorkflowID int

	var ensureSameTou *metadata_models.PolicySet

	for originDatasetAccession := range originDatasetImages {
		originDataset, err := tx.GetOriginDataset(ctx, originDatasetAccession)
		if err != nil {
			span.Error("failed to get origin dataset", err, slog.String("accession", originDatasetAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDataset == nil {
			span.Error("failed to find origin dataset", errors.New("failed to find origin dataset"), slog.String("accession", originDatasetAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		originDatasets[originDatasetAccession] = originDataset

		if ensureSameWorkflowID == 0 {
			ensureSameWorkflowID = originDataset.RemsWorkflowID
		}

		if ensureSameTou == nil {
			ensureSameTou = originDataset.Policy
		}

		// If only images from one dataset no need to compare
		if len(originDatasetImages) == 1 {
			continue
		}

		if ensureSameWorkflowID != originDataset.RemsWorkflowID {
			span.Info("user tried to combine datasets with different workflows id")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets originating from different DaCs"))
		}

		if !ensureSameTou.Equal(originDataset.Policy) {
			span.Info("user tried to combine datasets with different terms of use")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets with different terms of use"))
		}
	}

	onDemandDataset := buildOnDemandDataset(ctx, originDatasets, originDatasetImages)
	onDemandDataset.RequestedByUser = user
	span.SetAttributes(attribute.String("on-demand-dataset-accession", onDemandDataset.Accession))

	if err := tx.InsertOnDemandDataset(ctx, onDemandDataset, imageAccessionHash); err != nil {
		span.Error("failed to insert on demand dataset", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for originAccession, imageAccessions := range originDatasetImages {
		for imageAccession := range imageAccessions {
			if err := tx.InsertOnDemandDatasetImage(ctx, onDemandDataset.Accession, imageAccession); err != nil {
				span.Error("failed to insert on demand dataset image", err, slog.String("origin-accession", originAccession), slog.String("image-accession", imageAccession))

				return nil, connect.NewError(connect.CodeInternal, nil)
			}
		}
	}

	if err := on_demand_dataset_metadata_file_handler.RegisterOnDemandDataset(ctx, onDemandDataset); err != nil {
		span.Error("failed to register on demand dataset", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	// For now, we risk failing to commit after having uploaded and triggered ingestion for the metadata files(done by RegisterOnDemandDataset), but should be ok for now
	if err := tx.Commit(); err != nil {
		span.Error("failed to commit database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	span.Info("on demand dataset registered successfully")

	return connect.NewResponse(&dodservice.RequestDatasetCreationResponse{
		OnDemandDatasetAccession: onDemandDataset.Accession,
	}), nil
}

func (d *serviceImpl) GetOnDemandDatasetStatus(ctx context.Context, c *connect.Request[dodservice.GetOnDemandDatasetStatusRequest]) (*connect.Response[dodservice.GetOnDemandDatasetStatusResponse], error) {
	ctx, span := observability.StartSpan(ctx, "GetOnDemandDatasetStatus", attribute.String("accession", c.Msg.GetOnDemandDatasetAccession()))
	defer span.End()

	if c.Msg.GetOnDemandDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("empty dod dataset accession"))
	}

	onDemandDatasetReleased, err := database.IsOnDemandDatasetReleased(ctx, c.Msg.GetOnDemandDatasetAccession())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		span.Error("failed to check if on demand dataset is released", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	res := dodservice.GetOnDemandDatasetStatusResponse{
		Status: dodservice.GetOnDemandDatasetStatusResponse_STATUS_CREATING,
	}

	if onDemandDatasetReleased {
		res.Status = dodservice.GetOnDemandDatasetStatusResponse_STATUS_RELEASED
	}

	return connect.NewResponse(&res), nil
}

// hashImageAccessions takes a slice of image accessions, sorts them, removes duplicates and returns a sha256 hash of the slice
func hashImageAccessions(ids []string) string {
	sorted := append([]string(nil), ids...)
	slices.Sort(sorted)

	n := 0
	for _, id := range sorted {
		if n == 0 || id != sorted[n-1] {
			sorted[n] = id
			n++
		}
	}
	sorted = sorted[:n]

	h := sha256.New()
	for _, id := range sorted {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0}) // separator
	}

	return hex.EncodeToString(h.Sum(nil))
}
