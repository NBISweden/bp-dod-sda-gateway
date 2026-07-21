package observability

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	oteltracenoop "go.opentelemetry.io/otel/trace/noop"
)

var tracerName string

var promSrv *http.Server

func Tracer() oteltrace.Tracer {
	if tracerName == "" || !enabled {
		return oteltracenoop.NewTracerProvider().Tracer("noop")
	}

	return otel.GetTracerProvider().Tracer(tracerName)
}

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	tracerName = serviceName
	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil

		return err
	}
	if !enabled {
		return shutdown, nil
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	traceExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())
	if err != nil {
		handleErr(err)

		return
	}

	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
	if err != nil {
		handleErr(err)

		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	reg := prometheus.NewRegistry()
	metricsExposer, err := otelprom.New(
		otelprom.WithRegisterer(reg), // register exporter with this registry
	)
	if err != nil {
		handleErr(err)

		return
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(metricsExposer.Reader))
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	if err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		handleErr(err)

		return
	}

	prometheusMux := http.NewServeMux()

	prometheusMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	promSrv = &http.Server{
		Addr:              ":9090",
		Handler:           prometheusMux,
		ReadHeaderTimeout: 20 * time.Second,
	}
	go func() {
		if err := promSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start prometheus metrics server")
		}
	}()
	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		return promSrv.Shutdown(ctx)
	})

	return
}
