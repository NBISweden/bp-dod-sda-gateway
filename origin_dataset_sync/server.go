package origin_dataset_sync

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	privatedodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1/v1connect"
	publicdodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/public/v1/v1connect"
)

type Server struct {
	srv *http.Server
}

type Config struct {
	ConnectRPCInterceptors []connect.Interceptor
}

func (d *Server) ListenAndServe() error {
	return d.srv.ListenAndServe()
}
func (d *Server) Shutdown(ctx context.Context) error {
	return d.srv.Shutdown(ctx)
}

func NewServer(config Config) (*Server, error) {
	mux := http.NewServeMux()
	mux.Handle(privatedodservice.NewOriginDatasetSyncServiceHandler(
		&serviceImpl{
			syncer: ods,
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
			Addr:              fmt.Sprintf(":%d", servicePort),
			Handler:           mux,
			Protocols:         p,
			ReadHeaderTimeout: 20 * time.Second,
		},
	}

	return doDServer, nil
}
