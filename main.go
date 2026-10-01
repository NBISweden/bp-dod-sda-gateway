package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/NBISweden/bp-dod-sda-gateway/config"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/NBISweden/bp-dod-sda-gateway/database/postgres"
	"github.com/NBISweden/bp-dod-sda-gateway/dataset_on_demand"
	"github.com/NBISweden/bp-dod-sda-gateway/dataset_on_demand_admin"
	"github.com/NBISweden/bp-dod-sda-gateway/on_demand_dataset_metadata_file_handler"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader/metadata_submitter"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/auth_interceptor"
	configpkg "github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
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

	shutdown, err := observability.SetupOTelSDK(ctx, "bp-dod-sda-gateway")
	if err != nil {
		return fmt.Errorf("failed to setup OTel SDK: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("failed to shutdown OTel SDK", "err", err)
		}
	}()

	if err := postgres.InitPostgresSQLDatabase(); err != nil {
		return fmt.Errorf("failed to init postgres: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		return fmt.Errorf("failed to init otel connect interceptor: %w", err)
	}

	if err := on_demand_dataset_metadata_file_handler.Init(ctx); err != nil {
		return fmt.Errorf("failed to init dod metadata file handler: %w", err)
	}

	msdb, err := metadata_submitter.NewMetadataSubmitterDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to init metadata submitter database: %w", err)
	}
	defer func() {
		_ = msdb.Close()
	}()

	authenticator, err := auth_interceptor.NewAuthenticator(config.DodServiceJwtPubKeyUrl())
	if err != nil {
		return fmt.Errorf("failed to init authenticator: %w", err)
	}
	authMiddleware := authn.NewMiddleware(
		authenticator.Authenticate,
	)

	dodServer, err := dataset_on_demand.NewServer(
		dataset_on_demand.Config{
			Port:                   config.DodServicePort(),
			AuthMiddleware:         authMiddleware,
			ConnectRPCInterceptors: []connect.Interceptor{otelInterceptor},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to init dod service impl: %w", err)
	}
	dodAdminServer, err := dataset_on_demand_admin.NewServer(
		dataset_on_demand_admin.Config{
			Port:                    config.DodServicePort(),
			ConnectRPCInterceptors:  []connect.Interceptor{otelInterceptor},
			OriginDatasetFileLoader: msdb,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to init dod service impl: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := dodServer.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		}
	}()

	defer func() {
		serverShutdownCtx, serverShutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := dodServer.Shutdown(serverShutdownCtx); err != nil {
			slog.Error("failed to close http/https server", "error", err)
		}
		serverShutdownCancel()
	}()

	adminServerErr := make(chan error, 1)

	go func() {
		if err := dodAdminServer.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				adminServerErr <- err
			}
		}
	}()

	defer func() {
		serverShutdownCtx, serverShutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := dodAdminServer.Shutdown(serverShutdownCtx); err != nil {
			slog.Error("failed to close http/https server", "error", err)
		}
		serverShutdownCancel()
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	slog.Info("bp-dod-sda-gateway started",
		"dod-port", config.DodServicePort(),
		"dod-admin-port", config.DodAdminServicePort(),
	)

	select {
	case sig := <-sigc:
		slog.Info("received signal", slog.String("signal", sig.String()))

		return nil
	case err := <-adminServerErr:
		return err
	case err := <-serverErr:
		return err
	}
}
