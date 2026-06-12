package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/otelconnect"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/config"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/database"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/database/postgres"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/dod_metadata_file_handler"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/dod_service_impl"
	dodservice "github.com/imi-bigpicture/bp-dod-sda-gateway/grpc_gen/go/v1/v1connect"
	configpkg "github.com/imi-bigpicture/bp-dod-sda-gateway/internal/config"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/observability"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/origin_dataset_file_loader/metadata_submitter_database"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := configpkg.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	callbacks, err := observability.SetupOTelSDK(ctx, "bp-dod-sda-gateway")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer func() {
		if err := callbacks(ctx); err != nil {
			slog.Error("failed to shutdown observability", "error", err.Error())
		}
	}()

	if err := postgres.InitPostgresSQLDatabase(); err != nil {
		return fmt.Errorf("failed to init postgres: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	mux := http.NewServeMux()
	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		return fmt.Errorf("failed to init otel connect interceptor: %w", err)
	}

	if err := dod_metadata_file_handler.Init(ctx); err != nil {
		return fmt.Errorf("failed to init dod metadata file handler: %w", err)
	}

	msdb, err := metadata_submitter_database.NewMetadataSubmitterDatabase()
	if err != nil {
		return fmt.Errorf("failed to init metadata submitter database: %w", err)
	}
	defer func() {
		_ = msdb.Close()
	}()

	dodServiceImpl, err := dod_service_impl.NewDodServiceImpl(
		dod_service_impl.OriginDatasetFileLoader(msdb),
	)
	if err != nil {
		return fmt.Errorf("failed to init dod service impl: %w", err)
	}

	mux.Handle(dodservice.NewDatasetOnDemandServiceHandler(
		dodServiceImpl,
		connect.WithInterceptors(otelInterceptor),
	))

	reflector := grpcreflect.NewStaticReflector(
		dodservice.DatasetOnDemandServiceName,
	)

	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	p := new(http.Protocols)
	p.SetHTTP1(true)
	// Use h2c so we can serve HTTP/2 without TLS.
	p.SetUnencryptedHTTP2(true)
	s := http.Server{
		Addr:      fmt.Sprintf(":%d", config.DodServicePort()),
		Handler:   mux,
		Protocols: p,
	}

	if err := s.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// TODO sigterm handling

	return nil
}
