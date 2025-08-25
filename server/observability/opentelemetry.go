package observability

import (
	"context"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
)

func InitOpenTelemetry(ctx context.Context) error {
	var (
		metricExporter sdkmetric.Exporter
		traceExporter  sdktrace.SpanExporter
		err            error
	)

	grpcEndpoint := os.Getenv("QUEUETY_OTEL_OTLP_GRPC_ENDPOINT")
	httpEndpoint := os.Getenv("QUEUETY_OTEL_OTLP_HTTP_ENDPOINT")

	if grpcEndpoint != "" {

		conn, err := grpc.NewClient(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("error connecting to opentelemetry grpc collector: %v\n", err)
			return err
		}

		metricExporter, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			log.Printf("error creating grpc metrics exporter: %v\n", err)
			return err
		}

		traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			log.Printf("error creating grpc trace exporter: %v\n", err)
			return err
		}
	} else if httpEndpoint != "" {

		metricExporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(strings.TrimPrefix(httpEndpoint, "http://")),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			log.Printf("error creating http metrics exporter: %v\n", err)
			return err
		}

		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(strings.TrimPrefix(httpEndpoint, "http://")),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			log.Printf("error creating http trace exporter: %v\n", err)
			return err
		}
	} else {
		log.Printf("no opentelemetry endpoints configured %v\n", nil)
		return nil
	}

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	err = InitOTelMetrics()
	if err != nil {
		log.Printf("error initializing opentelemetry metrics: %v\n", err)
		return err
	}

	return nil
}

func ShutdownOpenTelemetry(ctx context.Context) error {
	var errs []error

	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if err := CloseMetrics(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

func GetMeter(name string) metric.Meter {
	return otel.GetMeterProvider().Meter(name)
}
