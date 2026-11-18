package origin_dataset_sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

type syncer interface {
	triggerSync(ctx context.Context)
	syncOriginDataset(ctx context.Context, datasetAccession string) error
}

type originDatasetSync struct {
	sync.RWMutex

	httpClient *http.Client

	syncingInProgress bool

	ctx    context.Context
	cancel context.CancelFunc

	manualSyncChan chan context.Context
}

var ods = &originDatasetSync{
	manualSyncChan: make(chan context.Context),
	httpClient: &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	},
}

func Init(ctx context.Context) error {
	ods.ctx, ods.cancel = context.WithCancel(ctx)

	go ods.syncer()

	return nil
}

func (ods *originDatasetSync) triggerSync(ctx context.Context) {
	if ods.syncingInProgress {
		return
	}
	ods.manualSyncChan <- ctx
}

type listSyncDatasetsResponse struct {
	Submissions []submission `json:"submissions"`
}
type submission struct {
	SubmissionId string `json:"submissionId"`
	Published    string `json:"published"`
	Ingested     string `json:"ingested"`
}

func (ods *originDatasetSync) syncer() {
	defer ods.cancel()
	for {
		syncCtx := ods.ctx
		select {
		case <-ods.ctx.Done():
			slog.Info("origin dataset syncer stopped - context done")

			return
		case manualSyncCtx := <-ods.manualSyncChan:
			syncCtx = manualSyncCtx
		case <-time.After(syncInterval):
		}
		ods.syncingInProgress = true
		ods.syncOriginDatasets(syncCtx)
		ods.syncingInProgress = false
	}
}
func (ods *originDatasetSync) syncOriginDatasets(ctx context.Context) {
	ctx, span := observability.StartSpan(ctx, "syncOriginDatasets")
	defer span.End()

	start, err := database.GetOriginDatasetSyncCursor(ctx)
	if err != nil {
		span.Error("failed to get origin dataset sync cursor from database", err)

		return
	}

	end := time.Now()

	syncUrl, err := url.JoinPath(metadataSubmitterURL, "sync")
	if err != nil {
		span.Error("failed to build sync API path", err)

		return
	}

	query := url.Values{}
	query.Set("publishedStart", start.UTC().Format(time.RFC3339Nano))
	query.Set("publishedEnd", end.UTC().Format(time.RFC3339Nano))
	if enc := query.Encode(); enc != "" {
		syncUrl += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncUrl, nil)
	if err != nil {
		span.Error("failed to build http request", err)

		return
	}
	token, err := ods.generateAndSignToken()
	if err != nil {
		span.Error("failed to generate and sign token", err)

		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rsp, err := ods.httpClient.Do(req)
	if err != nil {
		span.Error("failed to do http request", err)

		return
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		span.Error("non 200 status code when listing submissions to be synced", nil)

		return
	}

	parsedRsp := new(listSyncDatasetsResponse)

	if err := json.NewDecoder(rsp.Body).Decode(&parsedRsp); err != nil {
		span.Error("failed to decode response body", err)

		return
	}

	for _, submission := range parsedRsp.Submissions {
		if err := ods.syncOriginDataset(ctx, submission.SubmissionId); err != nil {
			span.Error("failed to sync submission", err, slog.String("submission-id", submission.SubmissionId))

			return
		}
	}

	if err := database.SetOriginDatasetSyncCursor(ctx, end); err != nil {
		span.Error("failed to set origin dataset sync cursor", err)
	}
}
func (ods *originDatasetSync) syncOriginDataset(ctx context.Context, datasetAccession string) error {
	ctx, span := observability.StartSpan(ctx, "syncOriginDataset", attribute.String("dataset-accession", datasetAccession))
	defer span.End()

	// TODO md standard v3

	od, err := ods.getOriginDatasetMetadata(ctx, datasetAccession)
	if err != nil {
		return fmt.Errorf("failed to get origin dataset metadata: %w", err)
	}

	if od.Policy == nil || len(od.Policy.Policies) != 1 {
		span.Warn("policy.xml not found or unexpected amount, skipping dataset")

		return nil
	}

	typeOfDataset, ok := od.Policy.Policies[0].Attributes.Get("type_of_dataset").(*metadata_models.NullableString)
	if !ok || typeOfDataset == nil || typeOfDataset.Value == nil {
		span.Warn("type_of_dataset not known, skipping dataset")

		return nil
	}

	if !strings.Contains(*typeOfDataset.Value, "Non-Clinical") {
		span.Info("type_of_dataset not supported, skipping dataset", slog.String("type_of_dataset", *typeOfDataset.Value))

		return nil
	}

	if od.RemsWorkflowID == -1 || od.RemsOrganisationID == "" {
		span.Warn("rems.xml not found, skipping dataset")

		return nil
	}

	// Check that required xml files are present
	if od.Dataset == nil {
		span.Warn("dataset.xml not found, skipping dataset")

		return nil
	}
	if od.Image == nil {
		span.Warn("image.xml not found, skipping dataset")

		return nil
	}
	if od.Observation == nil {
		span.Warn("observation.xml not found, skipping dataset")

		return nil
	}
	if od.Sample == nil {
		span.Warn("sample.xml not found, skipping dataset")

		return nil
	}
	if od.Staining == nil {
		span.Warn("staining.xml not found, skipping dataset")

		return nil
	}

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			span.Error("failed to rollback database transaction", err)
		}
	}()

	// TODO, answer should updates of origin dataset update any existing dataset on demand

	if err := tx.UpsertOriginDataset(ctx, &od.OriginDataset); err != nil {
		return fmt.Errorf("failed to insert dataset to database: %w", err)
	}

	for _, image := range od.Image.Images {
		if image.Accession == "" {
			span.Warn("image does not have an accession id, skipping dataset", slog.String("image-alias", image.Alias))

			return nil
		}

		if err := tx.UpsertDatasetImage(ctx, od.Accession, image.Accession); err != nil {
			return fmt.Errorf("failed to insert dataset image: %s to database: %w", image.Accession, err)
		}
		for _, file := range image.Files.Files {
			var found bool
			for _, fi := range od.files {
				if strings.HasSuffix(strings.TrimSuffix(fi.Path, ".c4gh"), strings.TrimSuffix(file.Filename, ".c4gh")) {
					found = true
					if err := tx.UpsertImageFile(ctx, od.Accession, image.Accession, fi.Accession, filepath.Base(file.Filename)); err != nil {
						return fmt.Errorf("failed to insert image: %s file: %s to database: %w", image.Accession, fi.Accession, err)
					}

					break
				}
			}
			if !found {
				span.Warn("accession id for image file not found, skipping dataset", slog.String("image-file", file.Filename))

				return nil
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit database transaction: %w", err)
	}

	return nil
}
