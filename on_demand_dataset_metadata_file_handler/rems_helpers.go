package on_demand_dataset_metadata_file_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
)

type resourcesCreateReq struct {
	ResourceID   string       `json:"resid"`
	Organization organization `json:"organization"`
	Licenses     []string     `json:"licenses"`
}
type catalogueItemsCreateReq struct {
	ResourceID    int           `json:"resid"`
	WorkflowID    int           `json:"wfid"`
	Organization  organization  `json:"organization"`
	Localizations localizations `json:"localizations"`
	Categories    []categories  `json:"categories,omitempty"`
}
type categories struct {
	CategoryId int `json:"category/id"`
}
type localizations struct {
	En localizationTitle `json:"en"`
}
type localizationTitle struct {
	Title string `json:"title"`
}
type organization struct {
	OrganizationID string `json:"organization/id"`
}

func (dmfh *dodMetadataFileHandler) doRemsRequest(_ context.Context, req *http.Request) ([]byte, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rems-Api-Key", remsKey)
	req.Header.Set("X-Rems-User-Id", remsUser)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
func (dmfh *dodMetadataFileHandler) createRemsResource(ctx context.Context, remsMetadata *metadata_models.RemsSet, datasetAccession string) (int, error) {
	ctx, span := observability.Tracer().Start(ctx, "createRemsResource")
	defer span.End()

	// check if rems resource already exists first
	getEndpoint, err := url.JoinPath(remsUrl, "api", "resources")
	if err != nil {
		return -1, fmt.Errorf("invalid base URL when building resources get endpoint: %w", err)
	}
	query := url.Values{}
	query.Set("resid", datasetAccession)
	if enc := query.Encode(); enc != "" {
		getEndpoint += "?" + enc
	}

	resourceGetRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, getEndpoint, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to build resources get request: %w", err)
	}

	resourcesGetResponseBody, err := dmfh.doRemsRequest(ctx, resourceGetRequest)
	if err != nil {
		return -1, fmt.Errorf("failed to do request to rems: %w", err)
	}

	var resourcesGetParsedResp []struct {
		// Only includes relevant fields
		ID int `json:"id"`
	}

	if err := json.Unmarshal(resourcesGetResponseBody, &resourcesGetParsedResp); err != nil {
		return -1, fmt.Errorf("failed to unmarshal resources get response body: %w", err)
	}

	if len(resourcesGetParsedResp) > 0 {
		return resourcesGetParsedResp[0].ID, nil
	}

	createReq := resourcesCreateReq{
		ResourceID: datasetAccession,
	}
	// TODO how to create resource from multiple orgs???
	for _, rems := range remsMetadata.Rems {
		createReq.Organization.OrganizationID = rems.OrganisationId
	}
	// Override values from rems.xml with hardcoded values when connected to a Demo rems instance
	if remsDemoOrganisationId != "" {
		createReq.Organization.OrganizationID = remsDemoOrganisationId
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		return -1, fmt.Errorf("failed to marshal resources create body: %w", err)
	}

	createEndpoint, err := url.JoinPath(remsUrl, "api", "resources", "create")
	if err != nil {
		return -1, fmt.Errorf("invalid base URL when building resources create endpoint: %w", err)
	}

	resourcesCreateRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, createEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return -1, fmt.Errorf("failed to build resources reate request: %w", err)
	}

	resourcesCreateResponseBody, err := dmfh.doRemsRequest(ctx, resourcesCreateRequest)
	if err != nil {
		return -1, err
	}

	resourcesCreateParsedResp := new(struct {
		Id int `json:"id"`
	})

	if err := json.Unmarshal(resourcesCreateResponseBody, resourcesCreateParsedResp); err != nil {
		return -1, fmt.Errorf("failed to unmarshal resources create response body: %w", err)
	}

	return resourcesCreateParsedResp.Id, nil
}

func (dmfh *dodMetadataFileHandler) createRemsCatalogueItem(ctx context.Context, remsMetadata *metadata_models.RemsSet, datasetAccession string, remsResourceID int) error {
	ctx, span := observability.Tracer().Start(ctx, "createRemsCatalogueItem")
	defer span.End()

	catalogueItemsGetEndpoint, err := url.JoinPath(remsUrl, "api", "catalogue-items")
	if err != nil {
		return fmt.Errorf("invalid base URL when building catalogue items get endpoint: %w", err)
	}
	query := url.Values{}
	query.Set("resource", datasetAccession)
	if enc := query.Encode(); enc != "" {
		catalogueItemsGetEndpoint += "?" + enc
	}

	catalogueItemsGetRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogueItemsGetEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to build catalogue items get request: %w", err)
	}

	catalogueItemsGetResponseBody, err := dmfh.doRemsRequest(ctx, catalogueItemsGetRequest)
	if err != nil {
		return fmt.Errorf("failed to do request to rems: %w", err)
	}

	var catalogueItemsGetParsedResp []struct {
		// Only includes relevant fields
		ID int `json:"id"`
	}

	if err := json.Unmarshal(catalogueItemsGetResponseBody, &catalogueItemsGetParsedResp); err != nil {
		return fmt.Errorf("failed to unmarshal catalogue items get response body: %w", err)
	}

	if len(catalogueItemsGetParsedResp) > 0 {
		return nil
	}

	ciCreateReq := &catalogueItemsCreateReq{
		ResourceID: remsResourceID,
		Localizations: localizations{
			En: localizationTitle{
				Title: fmt.Sprintf("On Demand Dataset: %s", datasetAccession),
			},
		},
	}

	// TODO how to create resource from multiple orgs???
	for _, rems := range remsMetadata.Rems {
		ciCreateReq.Organization.OrganizationID = rems.OrganisationId
		workflowID, err := strconv.Atoi(rems.WorkflowId)
		if err != nil {
			return fmt.Errorf("failed to parse workflow id: %w", err)
		}
		ciCreateReq.WorkflowID = workflowID
	}

	// Override values from rems.xml with hardcoded values when connected to a Demo rems instance
	if remsDemoOrganisationId != "" {
		ciCreateReq.Organization.OrganizationID = remsDemoOrganisationId
	}
	if remsDemoWorkflowId != -1 {
		ciCreateReq.WorkflowID = remsDemoWorkflowId
	}

	if remsCategoryId != 0 {
		ciCreateReq.Categories = []categories{
			{
				CategoryId: remsCategoryId,
			},
		}
	}

	catalogueItemsRequestBody, err := json.Marshal(ciCreateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal catalogue items create request body: %w", err)
	}

	catalogueItemsCreateEndpoint, err := url.JoinPath(remsUrl, "api", "catalogue-items", "create")
	if err != nil {
		return fmt.Errorf("invalid base URL when building catalogue items create endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, catalogueItemsCreateEndpoint, bytes.NewBuffer(catalogueItemsRequestBody))
	if err != nil {
		return fmt.Errorf("failed to build catalogue items create request: %w", err)
	}

	if _, err := dmfh.doRemsRequest(ctx, req); err != nil {
		return err
	}

	return nil
}
