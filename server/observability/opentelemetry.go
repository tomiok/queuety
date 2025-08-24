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
	// Configuración global de OpenTelemetry
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
)

// InitOpenTelemetry inicializa los proveedores de OpenTelemetry
func InitOpenTelemetry(ctx context.Context) error {
	var (
		metricExporter sdkmetric.Exporter
		traceExporter  sdktrace.SpanExporter
		err            error
	)

	// Obtener endpoints de las variables de entorno
	grpcEndpoint := os.Getenv("QUEUETY_OTEL_OTLP_GRPC_ENDPOINT")
	httpEndpoint := os.Getenv("QUEUETY_OTEL_OTLP_HTTP_ENDPOINT")

	// Prioridad: si hay endpoint gRPC, se usa gRPC
	if grpcEndpoint != "" {
		// Configurar conexión gRPC
		conn, err := grpc.NewClient(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Error conectando al colector OpenTelemetry gRPC: %v", err)
			return err
		}

		// Configurar exportador de métricas gRPC
		metricExporter, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			log.Printf("Error creando exportador de métricas gRPC: %v", err)
			return err
		}

		// Configurar exportador de trazas gRPC
		traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			log.Printf("Error creando exportador de trazas gRPC: %v", err)
			return err
		}
	} else if httpEndpoint != "" {
		// Configurar exportador de métricas HTTP
		metricExporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(strings.TrimPrefix(httpEndpoint, "http://")),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			log.Printf("Error creando exportador de métricas HTTP: %v", err)
			return err
		}

		// Configurar exportador de trazas HTTP
		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(strings.TrimPrefix(httpEndpoint, "http://")),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			log.Printf("Error creando exportador de trazas HTTP: %v", err)
			return err
		}
	} else {
		// Si no hay endpoints configurados, no inicializar OpenTelemetry
		log.Println("No se configuraron endpoints de OpenTelemetry")
		return nil
	}

	// Crear proveedor de métricas
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	// Crear proveedor de trazas
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)

	// Configurar propagación global
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Inicializar métricas de OpenTelemetry
	err = InitOTelMetrics()
	if err != nil {
		log.Printf("Error inicializando métricas de OpenTelemetry: %v", err)
		return err
	}

	return nil
}

// ShutdownOpenTelemetry cierra los proveedores de OpenTelemetry
func ShutdownOpenTelemetry(ctx context.Context) error {
	var errs []error

	// Cerrar proveedor de métricas
	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	// Cerrar proveedor de trazas
	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	// Cerrar métricas específicas
	if err := CloseMetrics(ctx); err != nil {
		errs = append(errs, err)
	}

	// Si hay errores, devolver el primero
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// GetTracer obtiene un rastreador para un nombre de componente específico
func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// GetMeter obtiene un medidor para un nombre de componente específico
func GetMeter(name string) metric.Meter {
	return otel.GetMeterProvider().Meter(name)
}
