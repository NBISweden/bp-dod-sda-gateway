package dataset_on_demand

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	publicdodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/public/v1/v1connect"
)

type Server struct {
	srv *http.Server
}

type Config struct {
	Port                   int
	AuthMiddleware         *authn.Middleware
	ConnectRPCInterceptors []connect.Interceptor
}

func (d *Server) ListenAndServe() error {
	return d.srv.ListenAndServe()
}
func (d *Server) Shutdown(ctx context.Context) error {
	return d.srv.Shutdown(ctx)
}

func NewServer(config Config) (*Server, error) {
	if config.AuthMiddleware == nil {
		return nil, errors.New("no auth middleware provided")
	}
	if config.Port == 0 {
		return nil, errors.New("no port provided")
	}

	mux := http.NewServeMux()
	mux.Handle(publicdodservice.NewDatasetOnDemandServiceHandler(
		&serviceImpl{},
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
			Handler:           config.AuthMiddleware.Wrap(mux),
			Protocols:         p,
			ReadHeaderTimeout: 20 * time.Second,
		},
	}

	return doDServer, nil
}
