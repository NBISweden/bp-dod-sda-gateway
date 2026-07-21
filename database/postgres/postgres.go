package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"embed"

	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/XSAM/otelsql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Imported to support using the embed migrationsFS as source for migrations
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type pgDb struct {
	db                 *sql.DB
	config             *dbConfig
	preparedStatements map[string]*sql.Stmt
	schemaVersion      uint

	metricsReg metric.Registration
}

var queries = make(map[string]string)

func InitPostgresSQLDatabase(options ...func(config *dbConfig)) error {
	dbConf := globalConf.clone()

	for _, o := range options {
		o(dbConf)
	}

	pg := &pgDb{db: nil, config: dbConf}

	pqConnectConfig, err := pq.NewConnectorConfig(pg.config.buildPostgresConfig())
	if err != nil {
		return fmt.Errorf("failed to setup postgres connect config: %w", err)
	}

	pg.db = otelsql.OpenDB(pqConnectConfig)
	if err := pg.db.Ping(); err != nil {
		_ = pg.Close()

		return fmt.Errorf("failed to connect to database: %w", err)
	}

	pg.metricsReg, err = otelsql.RegisterDBStatsMetrics(pg.db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		_ = pg.Close()

		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		_ = pg.Close()

		return err
	}
	defer func() {
		_ = sourceDriver.Close()
	}()

	driver, err := postgres.WithInstance(pg.db, &postgres.Config{
		SchemaName: pg.config.schema,
	})
	if err != nil {
		_ = pg.Close()

		return fmt.Errorf("failed to create postgres migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"postgres",
		driver,
	)
	if err != nil {
		_ = pg.Close()

		return fmt.Errorf("failed to create migration client: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = pg.Close()

		return fmt.Errorf("migration failed: %w", err)
	}

	var dirty bool
	pg.schemaVersion, dirty, err = m.Version()
	if err != nil {
		_ = pg.Close()

		return fmt.Errorf("failed to get database schema version: %w", err)
	}
	if dirty {
		_ = pg.Close()

		return errors.New("dirty database schema")
	}

	// Prepare the statements from the queries
	pg.preparedStatements = make(map[string]*sql.Stmt)
	for queryName, query := range queries {
		preparedStmt, err := pg.db.Prepare(query)
		if err != nil {
			log.Errorf("failed to prepare query: %s, due to: %v", queryName, err)
			_ = pg.Close()

			return fmt.Errorf("failed to prepare query: %s, due to: %w", queryName, err)
		}
		pg.preparedStatements[queryName] = preparedStmt
	}

	pg.db.SetMaxIdleConns(pg.config.maxIdleConnections)
	pg.db.SetMaxOpenConns(pg.config.maxOpenConnections)
	pg.db.SetConnMaxIdleTime(pg.config.connectionMaxIdleTime)
	pg.db.SetConnMaxLifetime(pg.config.connectionMaxLifeTime)

	database.RegisterDatabase(pg)

	return nil
}

func (db *pgDb) Ping(ctx context.Context) error {
	if db == nil || db.db == nil {
		return errors.New("database not initialized")
	}

	return db.db.PingContext(ctx)
}
func (db *pgDb) SchemaVersion() (uint, error) {
	return db.schemaVersion, nil
}

// Close terminates the connection to the database
func (db *pgDb) Close() error {
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

func (db *pgDb) BeginTransaction(ctx context.Context) (database.Transaction, error) {
	if db == nil || db.db == nil {
		return nil, errors.New("database not initialized")
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &pgTx{
		tx:   tx,
		pgDb: db,
	}, nil
}

func (db *pgDb) getPreparedStmt(tx *sql.Tx, queryName string) (*sql.Stmt, error) {
	if db == nil || db.preparedStatements == nil {
		return nil, errors.New("database not initialized")
	}

	stmt := db.preparedStatements[queryName]
	if stmt == nil {
		return nil, fmt.Errorf("statement with name: %s not found", queryName)
	}

	if tx == nil {
		return stmt, nil
	}

	return tx.Stmt(stmt), nil
}

func (db *pgDb) InsertOriginDataset(ctx context.Context, originDataset *models.OriginDataset) error {
	return db.insertOriginDataset(ctx, nil, originDataset)
}

func (db *pgDb) InsertDatasetImage(ctx context.Context, datasetAccession, imageAccession string) error {
	return db.insertDatasetImage(ctx, nil, datasetAccession, imageAccession)
}

func (db *pgDb) InsertImageFile(ctx context.Context, datasetAccession, imageAccession, fileAccession, baseFileName string) error {
	return db.insertImageFile(ctx, nil, datasetAccession, imageAccession, fileAccession, baseFileName)
}

func (db *pgDb) GetOriginDatasetAccessionFromImageAccession(ctx context.Context, imageAccession string) (string, error) {
	return db.getOriginDatasetAccessionFromImageAccession(ctx, nil, imageAccession)
}

func (db *pgDb) GetOriginDataset(ctx context.Context, accession string) (*models.OriginDataset, error) {
	return db.getOriginDataset(ctx, nil, accession)
}

func (db *pgDb) InsertOnDemandDataset(ctx context.Context, onDemandDataset *models.OnDemandDataset) error {
	return db.insertOnDemandDataset(ctx, nil, onDemandDataset)
}

func (db *pgDb) InsertOnDemandDatasetImage(ctx context.Context, onDemandDatasetAccession, imageAccession string) error {
	return db.insertOnDemandDatasetImage(ctx, nil, onDemandDatasetAccession, imageAccession)
}
func (db *pgDb) ListUnreleasedOnDemandDatasetMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error) {
	return db.listUnreleasedOnDemandDatasetMetadataFileAccessions(ctx, nil)
}

func (db *pgDb) ListOnDemandDatasetImageFiles(ctx context.Context, onDemandDatasetAccession string) (map[string]string, error) {
	return db.listOnDemandDatasetImageFiles(ctx, nil, onDemandDatasetAccession)
}

func (db *pgDb) SetOnDemandDatasetReleased(ctx context.Context, onDemandDatasetAccession string) error {
	return db.setOnDemandDatasetReleased(ctx, nil, onDemandDatasetAccession)
}

func (db *pgDb) IsOnDemandDatasetReleased(ctx context.Context, onDemandDatasetAccession string) (bool, error) {
	return db.isOnDemandDatasetReleased(ctx, nil, onDemandDatasetAccession)
}
func (db *pgDb) GetOnDemandDatasetRemsMetadata(ctx context.Context, onDemandDatasetAccession string) (*metadata_models.RemsSet, error) {
	return db.getOnDemandDatasetRemsMetadata(ctx, nil, onDemandDatasetAccession)
}
