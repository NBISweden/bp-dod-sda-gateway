package metadata_submitter

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/XSAM/otelsql"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type metadataSubmitterClient struct {
	client *http.Client

	ctx    context.Context
	cancel context.CancelFunc
	cache  map[string]*cacheEntry
	sync.RWMutex

	db                 *sql.DB
	config             *dbConfig
	preparedStatements map[string]*sql.Stmt

	metricsReg metric.Registration
}

type cacheEntry struct {
	lastAccessed  time.Time
	originDataset *models.OriginDataset
}

var queries = make(map[string]string)

func NewMetadataSubmitterDatabase(ctx context.Context) (origin_dataset_file_loader.OriginDatasetFileLoader, error) {
	ctx, cancel := context.WithCancel(ctx)
	dbConf := globalConf.clone()

	pg := &metadataSubmitterClient{
		db:     nil,
		config: dbConf,
		client: http.DefaultClient,
		ctx:    ctx,
		cancel: cancel,
	}

	pqConnectConfig, err := pq.NewConnectorConfig(pg.config.buildPostgresConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to setup postgres connect config: %w", err)
	}

	pg.db = otelsql.OpenDB(pqConnectConfig)
	if err := pg.db.Ping(); err != nil {
		_ = pg.Close()

		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	pg.metricsReg, err = otelsql.RegisterDBStatsMetrics(pg.db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		_ = pg.Close()

		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Prepare the statements from the queries
	pg.preparedStatements = make(map[string]*sql.Stmt)
	for queryName, query := range queries {
		preparedStmt, err := pg.db.Prepare(query)
		if err != nil {
			log.Errorf("failed to prepare query: %s, due to: %v", queryName, err)
			_ = pg.Close()

			return nil, fmt.Errorf("failed to prepare query: %s, due to: %w", queryName, err)
		}
		pg.preparedStatements[queryName] = preparedStmt
	}

	pg.db.SetMaxIdleConns(pg.config.maxIdleConnections)
	pg.db.SetMaxOpenConns(pg.config.maxOpenConnections)
	pg.db.SetConnMaxIdleTime(pg.config.connectionMaxIdleTime)
	pg.db.SetConnMaxLifetime(pg.config.connectionMaxLifeTime)

	return pg, nil
}

func (mdc *metadataSubmitterClient) Ping(ctx context.Context) error {
	if mdc == nil || mdc.db == nil {
		return errors.New("database not initialized")
	}

	if err := mdc.db.PingContext(ctx); err != nil {
		return err
	}

	rsp, err := mdc.client.Get(path.Join(mdc.config.metadataSubmitterURL, "health"))
	if err != nil {
		return err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", rsp.Status)
	}

	return nil
}

func (mdc *metadataSubmitterClient) cacheCleanTicker() {
	var done bool
	ticker := time.NewTicker(30 * time.Minute)
	for {
		select {
		case <-ticker.C:
		case <-mdc.ctx.Done():
			done = true
		}

		if done {
			break
		}

		cacheTTL := time.Minute * 30
		mdc.Lock()
		for k, v := range mdc.cache {
			if v.lastAccessed.Add(cacheTTL).Before(time.Now()) {
				delete(mdc.cache, k)
			}
		}
		mdc.Unlock()
	}
}

// Close terminates the connection to the database
func (mdc *metadataSubmitterClient) Close() error {
	if mdc == nil {
		return nil
	}

	var err error
	if mdc.preparedStatements != nil {
		for queryName, stmt := range mdc.preparedStatements {
			if stmtErr := stmt.Close(); stmtErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to close %s stmt, due to: %w", queryName, stmtErr))
			}
		}
	}

	if mdc.db != nil {
		err = errors.Join(err, mdc.db.Close())
	}

	if mdc.metricsReg != nil {
		err = errors.Join(err, mdc.metricsReg.Unregister())
	}

	mdc.cancel()

	return err
}

func (mdc *metadataSubmitterClient) getPreparedStmt(queryName string) (*sql.Stmt, error) {
	if mdc == nil || mdc.preparedStatements == nil {
		return nil, errors.New("database not initialized")
	}

	stmt := mdc.preparedStatements[queryName]
	if stmt == nil {
		return nil, fmt.Errorf("statement with name: %s not found", queryName)
	}

	return stmt, nil
}

func (mdc *metadataSubmitterClient) getDatasetMetadata(ctx context.Context, datasetAccession string) (*cacheEntry, error) {
	mdc.RLock()
	defer mdc.RUnlock()

	entry, ok := mdc.cache[datasetAccession]
	if ok {
		entry.lastAccessed = time.Now()

		return entry, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path.Join(mdc.config.metadataSubmitterURL, "sync", datasetAccession), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}
	token, err := mdc.generateAndSignToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate and sign token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rsp, err := mdc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do http request: %w", err)
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get dataset metadata unexpected status code: %s", rsp.Status)
	}

	zipData, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader from response body: %w", err)
	}

	originDataset := &models.OriginDataset{
		Accession: datasetAccession,
	}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from zip reader: %w", f.Name, err)
		}

		xmlDecoder := xml.NewDecoder(rc)
		switch f.Name {
		case "dataset.xml":
			err = xmlDecoder.Decode(&originDataset.Dataset)
		case "image.xml":
			err = xmlDecoder.Decode(&originDataset.Image)
		case "annotation.xml":
			err = xmlDecoder.Decode(&originDataset.Annotation)
		case "observation.xml":
			err = xmlDecoder.Decode(&originDataset.Observation)
		case "observer.xml":
			err = xmlDecoder.Decode(&originDataset.Observer)
		case "policy.xml":
			err = xmlDecoder.Decode(&originDataset.Policy)
		case "sample.xml":
			err = xmlDecoder.Decode(&originDataset.Sample)
		case "staining.xml":
			err = xmlDecoder.Decode(&originDataset.Staining)
		case "rems.xml":
			var remsEntry metadata_models.Rems
			if err = xmlDecoder.Decode(&remsEntry); err != nil {
				break
			}
			workflowID, err := strconv.Atoi(remsEntry.WorkflowId)
			if err != nil {
				return nil, fmt.Errorf("failed to parse rems workflow id to an integer: %w", err)
			}
			originDataset.RemsWorkflowID = workflowID
			originDataset.RemsOrganisationID = remsEntry.OrganisationId
		default:
		}

		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", f.Name, err)
		}
	}

	entry = &cacheEntry{
		lastAccessed:  time.Now(),
		originDataset: originDataset,
	}
	mdc.cache[datasetAccession] = entry

	return entry, nil
}

func (mdc *metadataSubmitterClient) generateAndSignToken() (string, error) {
	claims := map[string]any{
		jwt.ExpirationKey: time.Now().UTC().Add(15 * time.Second),
		jwt.IssuedAtKey:   time.Now().UTC(),
		jwt.IssuerKey:     "bp-dod-sda-gateway",
		jwt.SubjectKey:    "bp-dod-sda-gateway",
		jwt.AudienceKey:   "metadata-submitter",
	}

	jwtKey, err := jwk.ParseKey(mdc.config.privateKey, jwk.WithPEM(true))
	if err != nil {
		return "", err
	}
	if err := jwtKey.Set(jwk.AlgorithmKey, "ES256"); err != nil {
		return "", err
	}
	if err := jwk.AssignKeyID(jwtKey); err != nil {
		return "", err
	}

	token := jwt.New()
	for key, value := range claims {
		if err := token.Set(key, value); err != nil {
			return "", err
		}
	}

	tokenString, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, jwtKey))
	if err != nil {
		return "", err
	}

	return string(tokenString), nil
}
