package metadata_submitter_database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/XSAM/otelsql"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type metadataSubmitterPg struct {
	db                 *sql.DB
	config             *dbConfig
	preparedStatements map[string]*sql.Stmt

	metricsReg metric.Registration
}

var queries = make(map[string]string)

func NewMetadataSubmitterDatabase(options ...func(config *dbConfig)) (origin_dataset_file_loader.OriginDatasetFileLoader, error) {
	dbConf := globalConf.clone()

	for _, o := range options {
		o(dbConf)
	}

	pg := &metadataSubmitterPg{db: nil, config: dbConf}

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

func (db *metadataSubmitterPg) Ping(ctx context.Context) error {
	if db == nil || db.db == nil {
		return errors.New("database not initialized")
	}

	return db.db.PingContext(ctx)
}

// Close terminates the connection to the database
func (db *metadataSubmitterPg) Close() error {
	if db == nil {
		return nil
	}

	var err error
	if db.preparedStatements != nil {
		for queryName, stmt := range db.preparedStatements {
			if stmtErr := stmt.Close(); stmtErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to close %s stmt, due to: %w", queryName, stmtErr))
			}
		}
	}

	if db.db != nil {
		err = errors.Join(err, db.db.Close())
	}

	if db.metricsReg != nil {
		err = errors.Join(err, db.metricsReg.Unregister())
	}

	return err
}

func (db *metadataSubmitterPg) getPreparedStmt(queryName string) (*sql.Stmt, error) {
	if db == nil || db.preparedStatements == nil {
		return nil, errors.New("database not initialized")
	}

	stmt := db.preparedStatements[queryName]
	if stmt == nil {
		return nil, fmt.Errorf("statement with name: %s not found", queryName)
	}

	return stmt, nil
}

func (db *metadataSubmitterPg) UnmarshalFileToXml(ctx context.Context, file *origin_dataset_file_loader.FileInfo, dst any) error {
	return db.unmarshalFileToXml(ctx, file, dst)
}

func (db *metadataSubmitterPg) ListDatasetFiles(ctx context.Context, datasetAccession string) ([]*origin_dataset_file_loader.FileInfo, error) {
	return db.listDatasetFiles(ctx, datasetAccession)
}

func (db *metadataSubmitterPg) GetRemsWorkFlowIDAndOrganisationID(ctx context.Context, datasetAccession string) (int, string, error) {
	return db.getRemsWorkFlowIDAndOrganisationID(ctx, datasetAccession)
}
