package metadata_submitter

import (
	"fmt"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/lib/pq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type dbConfig struct {
	metadataSubmitterURL string
	privateKey           []byte

	host                  string
	port                  uint16
	user                  string
	password              string
	databaseName          string
	schema                string
	cACert                string
	sslMode               string
	clientCert            string
	clientKey             string
	maxIdleConnections    int
	maxOpenConnections    int
	connectionMaxIdleTime time.Duration
	connectionMaxLifeTime time.Duration
}

// Initialize globalConf with default values
var globalConf = &dbConfig{
	host:                  "", // No default, needs to be provided by config / InitPostgresSQLDatabase options
	port:                  5432,
	user:                  "", // No default, needs to be provided by config / InitPostgresSQLDatabase options
	password:              "", // No default, needs to be provided by config / InitPostgresSQLDatabase options
	databaseName:          "metadata-submitter",
	schema:                "public",
	cACert:                "",
	sslMode:               "prefer",
	clientCert:            "",
	clientKey:             "",
	maxIdleConnections:    2,
	maxOpenConnections:    20,
	connectionMaxIdleTime: 5 * time.Minute,
	connectionMaxLifeTime: 0,
}

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "metadata_submitter_sync_api.url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The url to the metadata submitter sync api")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				globalConf.metadataSubmitterURL = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_sync_api.private_key",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The private key used to sign token used when calling the metadata submitter sync API with ES256 algoritm")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.privateKey = []byte(viper.GetString(flagName))
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.host",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.host, "The host the metadata submitter postgres database is served on")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				globalConf.host = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Uint16(flagName, globalConf.port, "The port the metadata submitter database is served on")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.port = viper.GetUint16(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.user",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.user, "Username used to authenticate with in communication with metadata submitter database")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				globalConf.user = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.password",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.password, "Password used to authenticate with in communication with metadata submitter database")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				globalConf.password = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.ssl_mode",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.sslMode, "The metadata submitter database ssl mode")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.sslMode = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.ca_cert",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.cACert, "The metadata submitter database ca cert")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.cACert = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.client_cert",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.clientCert, "The cert the client will use in communication with the metadata submitter database")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.clientCert = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.client_key",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.clientKey, "The key for the client cert the client will use in communication with the metadata submitter database")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.clientKey = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.name",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.databaseName, "Metadata submitter database name to connect to")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.databaseName = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.schema",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, globalConf.schema, "Metadata submitter database schema to use as search path")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.schema = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.max_idle_connections",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, globalConf.maxIdleConnections, "Sets the maximum number of connections in the idle connection pool, if set to <= 0 no idle connections are retained.")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.maxIdleConnections = viper.GetInt(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.max_open_connections",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, globalConf.maxOpenConnections, "Sets the maximum number of open connections to the database, set to <= 0 for unlimited")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.maxOpenConnections = viper.GetInt(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.connection_max_idle_time",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Duration(flagName, globalConf.connectionMaxIdleTime, "Sets the maximum amount of time a connection may be idle, set to <= 0 for unlimited. Expects a go time.Duration parsable string")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.connectionMaxIdleTime = viper.GetDuration(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_database.connection_max_life_time",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Duration(flagName, globalConf.connectionMaxLifeTime, "Sets the maximum amount of time a connection may be reused, set to <= 0 for unlimited. Expects a go time.Duration parsable string")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				globalConf.connectionMaxLifeTime = viper.GetDuration(flagName)
			},
		},
	)
}

func (c *dbConfig) clone() *dbConfig {
	return &dbConfig{
		host:                  c.host,
		port:                  c.port,
		user:                  c.user,
		password:              c.password,
		databaseName:          c.databaseName,
		schema:                c.schema,
		cACert:                c.cACert,
		sslMode:               c.sslMode,
		clientCert:            c.clientCert,
		clientKey:             c.clientKey,
		maxIdleConnections:    c.maxIdleConnections,
		maxOpenConnections:    c.maxOpenConnections,
		connectionMaxIdleTime: c.connectionMaxIdleTime,
		connectionMaxLifeTime: c.connectionMaxLifeTime,
	}
}

// buildPostgresConfig builds a postgresql config source string to use with sql.OpenDB().
func (c *dbConfig) buildPostgresConfig() pq.Config {
	return pq.Config{
		Host:        c.host,
		Port:        c.port,
		Database:    c.databaseName,
		User:        c.user,
		Password:    c.password,
		SSLMode:     pq.SSLMode(c.sslMode),
		SSLCert:     c.clientCert,
		SSLKey:      c.clientKey,
		SSLRootCert: c.cACert,
		Options:     fmt.Sprintf("-c search_path=%s", c.schema),
	}
}
