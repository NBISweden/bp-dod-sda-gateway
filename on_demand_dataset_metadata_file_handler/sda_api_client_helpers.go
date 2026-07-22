package on_demand_dataset_metadata_file_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// fileInfo based on https://github.com/neicnordic/sensitive-data-archive/blob/main/sda/cmd/api/swagger_v1.yml#L505
// only relevant fields are specified
type fileInfo struct {
	FileId    string `json:"fileID"`
	InboxPath string `json:"inboxPath"`
	Status    string `json:"fileStatus"`
}

func (dmfh *dodMetadataFileHandler) triggerFileIngest(ctx context.Context, datasetAccession string, metadataFileType metadata_models.MetadataFileType) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerFileIngest", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("metadata-file-type", metadataFileType.String())))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "file", "ingest")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	ingestReq := struct {
		FilePath string `json:"filepath"`
		User     string `json:"user"`
	}{
		FilePath: fmt.Sprintf("%s/METADATA/%s.xml.c4gh", datasetAccession, metadataFileType.String()),
		User:     dmfh.uploadUser,
	}

	reqBody, err := json.Marshal(ingestReq)
	if err != nil {
		return fmt.Errorf("failed to marshal ingest body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerFileAccession(ctx context.Context, fileID, fileAccession string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerFileAccession", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "file", "accession")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	query := url.Values{}
	query.Set("fileid", fileID)
	query.Set("accessionid", fileAccession)

	if enc := query.Encode(); enc != "" {
		endpoint += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerDatasetCreation(ctx context.Context, datasetAccession string, datasetMetadataFiles map[metadata_models.MetadataFileType]string, imageAccessionFileNames map[string]map[string]string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerDatasetCreation", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("accession", datasetAccession)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "dataset", "create")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	var fileAccessions []string
	fileNames := make(map[string]string)

	for metadataType, metadataFileAccession := range datasetMetadataFiles {
		// Exclude rems
		if metadataType == metadata_models.MetadataFileTypeRems {
			continue
		}
		fileAccessions = append(fileAccessions, metadataFileAccession)
		fileNames[metadataFileAccession] = fmt.Sprintf("METADATA/%s.xml.c4gh", metadataType.String())
	}

	for imageAccession, imageFileNames := range imageAccessionFileNames {
		for fileAccession, baseFileName := range imageFileNames {
			fileAccessions = append(fileAccessions, fileAccession)
			fileNames[fileAccession] = fmt.Sprintf("IMAGES/IMAGE_%s/%s.c4gh", imageAccession, baseFileName)
		}
	}

	datasetCreateReq := struct {
		DatasetAccession  string            `json:"dataset_id"`
		FileAccessionIDs  []string          `json:"accession_ids"`
		User              string            `json:"user"`
		FileDownloadPaths map[string]string `json:"file_download_paths"`
	}{
		DatasetAccession:  datasetAccession,
		FileAccessionIDs:  fileAccessions,
		User:              dmfh.uploadUser,
		FileDownloadPaths: fileNames,
	}

	reqBody, err := json.Marshal(datasetCreateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerDatasetRelease(ctx context.Context, datasetAccession string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerDatasetRelease", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("accession", datasetAccession)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "dataset", "release", datasetAccession)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) listMetadataFiles(ctx context.Context, datasetAccession string) ([]*fileInfo, error) {
	ctx, span := observability.Tracer().Start(ctx, "listMetadataFiles", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "files")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return paginate(ctx, func(ctx context.Context, nextCursor string) ([]*fileInfo, string, error) {
		listUrl := endpoint
		query := url.Values{}
		query.Set("path_prefix", datasetAccession)
		if nextCursor != "" {
			query.Set("cursor", nextCursor)
		}
		if enc := query.Encode(); enc != "" {
			listUrl += "?" + enc
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, listUrl, nil)
		if err != nil {
			return nil, "", fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+inboxToken)

		resp, err := dmfh.httpClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("http request: %w", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)

			return nil, "", fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
		}

		var fileList []*fileInfo
		if err := json.NewDecoder(resp.Body).Decode(&fileList); err != nil {
			return nil, "", fmt.Errorf("failed to decode /files response: %w", err)
		}

		return fileList, resp.Header.Get("X-Next-Cursor"), nil
	})
}

func paginate[T any](
	ctx context.Context,
	fetch func(ctx context.Context, cursor string) ([]T, string, error),
) ([]T, error) {
	var all []T
	var nextCursor string // nil = first call
	seen := make(map[string]struct{})
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, next, err := fetch(ctx, nextCursor)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if next == "" {
			return all, nil
		}
		if _, dup := seen[next]; dup {
			return nil, fmt.Errorf("pagination aborted: server returned repeated pageToken %q", next)
		}
		seen[next] = struct{}{}
		nextCursor = next
	}
}
