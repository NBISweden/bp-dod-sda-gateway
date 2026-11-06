package dataset_on_demand_admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	privatedodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1/v1connect"
	publicdodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/public/v1/v1connect"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
)

type Server struct {
	srv *http.Server
}

type Config struct {
	Port                    int
	ConnectRPCInterceptors  []connect.Interceptor
	OriginDatasetFileLoader origin_dataset_file_loader.OriginDatasetFileLoader
}

func (d *Server) ListenAndServe() error {
	return d.srv.ListenAndServe()
}
func (d *Server) Shutdown(ctx context.Context) error {
	return d.srv.Shutdown(ctx)
}

func NewServer(config Config) (*Server, error) {
	if config.OriginDatasetFileLoader == nil {
		return nil, errors.New("no origin dataset file loader provided")
	}
	if config.Port == 0 {
		return nil, errors.New("no port provided")
	}

	mux := http.NewServeMux()
	mux.Handle(privatedodservice.NewDatasetOnDemandAdminServiceHandler(
		&serviceImpl{
			originDatasetFileLoader: config.OriginDatasetFileLoader,
		},
		connect.WithInterceptors(config.ConnectRPCInterceptors...),
	))

	reflector := grpcreflect.NewStaticReflector(
		publicdodservice.DatasetOnDemandServiceName,
	)

	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	p := new(http.Protocols)
	p.SetHTTP1(true)
	// Use h2c so we can serve HTTP/2 without TLS.
	p.SetUnencryptedHTTP2(true)
	doDServer := &Server{
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%d", config.Port),
			Handler:           mux,
			Protocols:         p,
			ReadHeaderTimeout: 20 * time.Second,
		},
	}

	return doDServer, nil
}
