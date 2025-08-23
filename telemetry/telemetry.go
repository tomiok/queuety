package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	meter   metric.Meter
	tracer  trace.Tracer
	metrics *Metrics
	enabled bool
}

type Config struct {
	Enabled  bool   `yaml:"enabled"`
	Backend  string `yaml:"backend"` // prometheus, datadog, otlp
	Endpoint string `yaml:"endpoint"`
	APIKey   string `yaml:"api_key"`
}

func New(cfg Config) (*Telemetry, error) {
	if !cfg.Enabled {
		return &Telemetry{enabled: false}, nil
	}

	// Setup OTEL provider based on backend
	if err := setupProvider(cfg); err != nil {
		return nil, err
	}

	meter := otel.Meter("queuety")
	tracer := otel.Tracer("queuety")

	metrics, err := newMetrics(meter)
	if err != nil {
		return nil, err
	}

	return &Telemetry{
		meter:   meter,
		tracer:  tracer,
		metrics: metrics,
		enabled: true,
	}, nil
}

// setupProvider configures the OpenTelemetry provider based on the backend
func setupProvider(cfg Config) error {
	// Setup metric provider
	metricProvider, err := setupMetricProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup metric provider: %w", err)
	}
	otel.SetMeterProvider(metricProvider)

	// Setup trace provider
	traceProvider, err := setupTraceProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup trace provider: %w", err)
	}
	otel.SetTracerProvider(traceProvider)

	return nil
}

func setupMetricProvider(cfg Config) (metric.MeterProvider, error) {
	switch cfg.Backend {
	case "prometheus":
		return setupPrometheusProvider()
	case "datadog":
		return setupDatadogProvider(cfg)
	case "otlp":
		return setupOTLPMetricProvider(cfg)
	default:
		return setupPrometheusProvider() // Default to Prometheus
	}
}

func setupTraceProvider(cfg Config) (trace.TracerProvider, error) {
	switch cfg.Backend {
	case "datadog":
		return setupDatadogTraceProvider(cfg)
	case "otlp":
		return setupOTLPTraceProvider(cfg)
	case "jaeger":
		return setupJaegerProvider(cfg)
	default:
		// For prometheus we can use console or no-op tracer
		return setupConsoleTraceProvider()
	}
}

func setupPrometheusProvider() (metric.MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	return provider, nil
}

func setupDatadogProvider(cfg Config) (metric.MeterProvider, error) {
	// Note: Datadog uses OTLP for metrics
	return setupOTLPMetricProvider(Config{
		Endpoint: "https://api.datadoghq.com/api/v2/otlp/v1/metrics",
		APIKey:   cfg.APIKey,
	})
}

func setupOTLPMetricProvider(cfg Config) (metric.MeterProvider, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:4318/v1/metrics" // Default OTLP HTTP endpoint
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.APIKey != "" {
		opts = append(opts, otlpmetrichttp.WithHeaders(map[string]string{
			"DD-API-KEY": cfg.APIKey, // For Datadog
		}))
	}

	exporter, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	return provider, nil
}

func setupDatadogTraceProvider(cfg Config) (trace.TracerProvider, error) {
	return setupOTLPTraceProvider(Config{
		Endpoint: "https://api.datadoghq.com/api/v2/otlp/v1/traces",
		APIKey:   cfg.APIKey,
	})
}

func setupOTLPTraceProvider(cfg Config) (trace.TracerProvider, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:4318/v1/traces"
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.APIKey != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
			"DD-API-KEY": cfg.APIKey,
		}))
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("queuety"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	return provider, nil
}

func setupJaegerProvider(cfg Config) (trace.TracerProvider, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:14268/api/traces"
	}

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("queuety"),
		)),
	)

	return provider, nil
}

func setupConsoleTraceProvider() (trace.TracerProvider, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("queuety"),
		)),
	)

	return provider, nil
}
