package observability

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	// Métricas de Prometheus
	MessagesPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_published_total",
			Help: "Total published messages by topic",
		},
		[]string{"topic"},
	)

	MessagesDeliveredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_delivered_total",
			Help: "Total delivered messages by topic",
		},
		[]string{"topic"},
	)

	MessagesFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_failed_total",
			Help: "Failed message deliveries by topic",
		},
		[]string{"topic"},
	)

	messageProcessingBuckets = []float64{
		0.25, // Bucket para P50 (50%)
		2.5,  // Bucket para P95 (95%)
		5,    // Bucket para P99 (99%)
		10,   // Bucket para P99.9 (99.9%)
	}

	MessageProcessingSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queuety_message_processing_seconds",
			Help:    "Processing latency histogram by topic and operation",
			Buckets: messageProcessingBuckets,
		},
		[]string{"topic", "operation"},
	)

	MessageProcessingAverage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queuety_message_processing_average_seconds",
			Help: "Average processing time for messages by topic and operation",
		},
		[]string{"topic", "operation"},
	)

	TopicsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queuety_topics_total",
			Help: "Number of active topics",
		},
	)

	SubscribersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queuety_subscribers_total",
			Help: "Subscribers per topic",
		},
		[]string{"topic"},
	)

	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queuety_active_connections",
			Help: "Current TCP connections count",
		},
	)

	BadgerOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_badger_operations_total",
			Help: "BadgerDB operation metrics",
		},
		[]string{"operation", "status"},
	)

	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_auth_attempts_total",
			Help: "Authentication attempts tracking",
		},
		[]string{"result"},
	)

	// Métricas de OpenTelemetry
	otelMeterProvider *sdkmetric.MeterProvider
	otelMeter         metric.Meter

	otelMessagesPublishedTotal   metric.Int64Counter
	otelMessagesDeliveredTotal   metric.Int64Counter
	otelMessagesFailedTotal      metric.Int64Counter
	otelMessageProcessingSeconds metric.Float64Histogram
	otelMessageProcessingAverage metric.Float64Gauge
	otelTopicsTotal              metric.Float64Gauge
	otelSubscribersTotal         metric.Float64Gauge
	otelActiveConnections        metric.Float64Gauge
	otelBadgerOperationsTotal    metric.Int64Counter
	otelAuthAttemptsTotal        metric.Int64Counter
)

// InitOTelMetrics inicializa los proveedores de métricas de OpenTelemetry
func InitOTelMetrics() error {
	otelMeterProvider = sdkmetric.NewMeterProvider()

	otelMeter = otelMeterProvider.Meter("queuety") // CREAR APP_NAME EN EL .env

	var err error

	// Crear contadores de OpenTelemetry
	otelMessagesPublishedTotal, err = otelMeter.Int64Counter("queuety_messages_published_total",
		metric.WithDescription("Total published messages by topic"),
	)
	if err != nil {
		log.Printf("Error creando contador de mensajes publicados: %v", err)
		return err
	}

	otelMessagesDeliveredTotal, err = otelMeter.Int64Counter("queuety_messages_delivered_total",
		metric.WithDescription("Total delivered messages by topic"),
	)
	if err != nil {
		log.Printf("Error creando contador de mensajes entregados: %v", err)
		return err
	}

	otelMessagesFailedTotal, err = otelMeter.Int64Counter("queuety_messages_failed_total",
		metric.WithDescription("Failed message deliveries by topic"),
	)
	if err != nil {
		log.Printf("Error creando contador de mensajes fallidos: %v", err)
		return err
	}

	otelMessageProcessingSeconds, err = otelMeter.Float64Histogram("queuety_message_processing_seconds",
		metric.WithDescription("Processing latency histogram by topic and operation"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		log.Printf("Error creando histograma de procesamiento de mensajes: %v", err)
		return err
	}

	otelMessageProcessingAverage, err = otelMeter.Float64Gauge("queuety_message_processing_average_seconds",
		metric.WithDescription("Average processing time for messages by topic and operation"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		log.Printf("Error creando gauge de promedio de procesamiento de mensajes: %v", err)
		return err
	}

	otelTopicsTotal, err = otelMeter.Float64Gauge("queuety_topics_total",
		metric.WithDescription("Number of active topics"),
	)
	if err != nil {
		log.Printf("Error creando gauge de temas totales: %v", err)
		return err
	}

	otelSubscribersTotal, err = otelMeter.Float64Gauge("queuety_subscribers_total",
		metric.WithDescription("Subscribers per topic"),
	)
	if err != nil {
		log.Printf("Error creando gauge de suscriptores totales: %v", err)
		return err
	}

	otelActiveConnections, err = otelMeter.Float64Gauge("queuety_active_connections",
		metric.WithDescription("Current TCP connections count"),
	)
	if err != nil {
		log.Printf("Error creando gauge de conexiones activas: %v", err)
		return err
	}

	otelBadgerOperationsTotal, err = otelMeter.Int64Counter("queuety_badger_operations_total",
		metric.WithDescription("BadgerDB operation metrics"),
	)
	if err != nil {
		log.Printf("Error creando contador de operaciones de BadgerDB: %v", err)
		return err
	}

	otelAuthAttemptsTotal, err = otelMeter.Int64Counter("queuety_auth_attempts_total",
		metric.WithDescription("Authentication attempts tracking"),
	)
	if err != nil {
		log.Printf("Error creando contador de intentos de autenticación: %v", err)
		return err
	}

	return nil
}

// CloseMetrics cierra los proveedores de métricas
func CloseMetrics(ctx context.Context) error {
	if otelMeterProvider != nil {
		return otelMeterProvider.Shutdown(ctx)
	}
	return nil
}

// IncrementPublishedMessages incrementa el contador de mensajes publicados para un tema específico
func IncrementPublishedMessages(topic string) {
	// Incrementar métrica de Prometheus
	MessagesPublishedTotal.WithLabelValues(topic).Inc()

	// Incrementar métrica de OpenTelemetry (si está inicializada)
	if otelMessagesPublishedTotal != nil {
		otelMessagesPublishedTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

// IncrementDeliveredMessages incrementa el contador de mensajes entregados para un tema específico
func IncrementDeliveredMessages(topic string) {
	MessagesDeliveredTotal.WithLabelValues(topic).Inc()

	if otelMessagesDeliveredTotal != nil {
		otelMessagesDeliveredTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

// IncrementFailedMessages incrementa el contador de mensajes fallidos para un tema específico
func IncrementFailedMessages(topic string) {
	MessagesFailedTotal.WithLabelValues(topic).Inc()

	if otelMessagesFailedTotal != nil {
		otelMessagesFailedTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

// ObserveMessageProcessingTime observa el tiempo de procesamiento de un mensaje
func ObserveMessageProcessingTime(topic, operation string, duration float64) {
	MessageProcessingSeconds.WithLabelValues(topic, operation).Observe(duration)
	MessageProcessingAverage.WithLabelValues(topic, operation).Set(duration)

	if otelMessageProcessingSeconds != nil {
		otelMessageProcessingSeconds.Record(context.Background(), duration, metric.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("operation", operation),
		))
	}

	if otelMessageProcessingAverage != nil {
		otelMessageProcessingAverage.Record(context.Background(), duration, metric.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("operation", operation),
		))
	}
}

// UpdateActiveTopicsCount actualiza el número de temas activos
func UpdateActiveTopicsCount(count int) {
	// Actualizar métrica de Prometheus
	TopicsTotal.Set(float64(count))

	if otelTopicsTotal != nil {
		otelTopicsTotal.Record(context.Background(), float64(count))
	}
}

// UpdateSubscribersCount actualiza el número de suscriptores para un tema específico
func UpdateSubscribersCount(topic string, count int) {
	// Actualizar métrica de Prometheus
	SubscribersTotal.WithLabelValues(topic).Set(float64(count))

	if otelSubscribersTotal != nil {
		otelSubscribersTotal.Record(context.Background(), float64(count), metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

// UpdateActiveConnectionsCount actualiza el número de conexiones TCP activas
func UpdateActiveConnectionsCount(count int) {
	// Actualizar métrica de Prometheus
	ActiveConnections.Set(float64(count))

	if otelActiveConnections != nil {
		otelActiveConnections.Record(context.Background(), float64(count))
	}
}

// IncrementBadgerOperation incrementa el contador de operaciones de BadgerDB
func IncrementBadgerOperation(operation, status string) {
	BadgerOperationsTotal.WithLabelValues(operation, status).Inc()

	if otelBadgerOperationsTotal != nil {
		otelBadgerOperationsTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		))
	}
}

// IncrementAuthAttempt incrementa el contador de intentos de autenticación
func IncrementAuthAttempt(result string) {
	AuthAttemptsTotal.WithLabelValues(result).Inc()

	if otelAuthAttemptsTotal != nil {
		otelAuthAttemptsTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("result", result),
		))
	}
}
