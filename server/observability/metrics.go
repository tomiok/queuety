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
		0.25,
		2.5,
		5,
		10,
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

func InitOTelMetrics() error {
	otelMeterProvider = sdkmetric.NewMeterProvider()

	otelMeter = otelMeterProvider.Meter(ServiceName)

	var err error

	otelMessagesPublishedTotal, err = otelMeter.Int64Counter("queuety_messages_published_total",
		metric.WithDescription("Total published messages by topic"),
	)
	if err != nil {
		log.Printf("error creating published messages counter: %v\n", err)
		return err
	}

	otelMessagesDeliveredTotal, err = otelMeter.Int64Counter("queuety_messages_delivered_total",
		metric.WithDescription("Total delivered messages by topic"),
	)
	if err != nil {
		log.Printf("error creating delivered messages counter: %v\n", err)
		return err
	}

	otelMessagesFailedTotal, err = otelMeter.Int64Counter("queuety_messages_failed_total",
		metric.WithDescription("Failed message deliveries by topic"),
	)
	if err != nil {
		log.Printf("error creating failed messages counter: %v\n", err)
		return err
	}

	otelMessageProcessingSeconds, err = otelMeter.Float64Histogram("queuety_message_processing_seconds",
		metric.WithDescription("Processing latency histogram by topic and operation"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		log.Printf("error creating message processing histogram: %v\n", err)
		return err
	}

	otelMessageProcessingAverage, err = otelMeter.Float64Gauge("queuety_message_processing_average_seconds",
		metric.WithDescription("Average processing time for messages by topic and operation"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		log.Printf("error creating message processing average gauge: %v\n", err)
		return err
	}

	otelTopicsTotal, err = otelMeter.Float64Gauge("queuety_topics_total",
		metric.WithDescription("Number of active topics"),
	)
	if err != nil {
		log.Printf("error creating total topics gauge: %v\n", err)
		return err
	}

	otelSubscribersTotal, err = otelMeter.Float64Gauge("queuety_subscribers_total",
		metric.WithDescription("Subscribers per topic"),
	)
	if err != nil {
		log.Printf("error creating total subscribers gauge: %v\n", err)
		return err
	}

	otelActiveConnections, err = otelMeter.Float64Gauge("queuety_active_connections",
		metric.WithDescription("Number of active TCP connections"),
	)
	if err != nil {
		log.Printf("error creating active connections gauge: %v\n", err)
		return err
	}

	otelBadgerOperationsTotal, err = otelMeter.Int64Counter("queuety_badger_operations_total",
		metric.WithDescription("Total BadgerDB operations"),
	)
	if err != nil {
		log.Printf("error creating BadgerDB operations counter: %v\n", err)
		return err
	}

	otelAuthAttemptsTotal, err = otelMeter.Int64Counter("queuety_auth_attempts_total",
		metric.WithDescription("Authentication attempts tracking"),
	)
	if err != nil {
		log.Printf("error creating authentication attempts counter: %v\n", err)
		return err
	}

	return nil
}

func CloseMetrics(ctx context.Context) error {
	if otelMeterProvider != nil {
		return otelMeterProvider.Shutdown(ctx)
	}
	return nil
}

func IncrementPublishedMessages(topic string) {
	MessagesPublishedTotal.WithLabelValues(topic).Inc()

	if otelMessagesPublishedTotal != nil {
		otelMessagesPublishedTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

func IncrementDeliveredMessages(topic string) {
	MessagesDeliveredTotal.WithLabelValues(topic).Inc()

	if otelMessagesDeliveredTotal != nil {
		otelMessagesDeliveredTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

func IncrementFailedMessages(topic string) {
	MessagesFailedTotal.WithLabelValues(topic).Inc()

	if otelMessagesFailedTotal != nil {
		otelMessagesFailedTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

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

func UpdateActiveTopicsCount(count int) {
	TopicsTotal.Set(float64(count))

	if otelTopicsTotal != nil {
		otelTopicsTotal.Record(context.Background(), float64(count))
	}
}

func UpdateSubscribersCount(topic string, count int) {
	SubscribersTotal.WithLabelValues(topic).Set(float64(count))

	if otelSubscribersTotal != nil {
		otelSubscribersTotal.Record(context.Background(), float64(count), metric.WithAttributes(
			attribute.String("topic", topic),
		))
	}
}

func UpdateActiveConnectionsCount(count int) {
	ActiveConnections.Set(float64(count))

	if otelActiveConnections != nil {
		otelActiveConnections.Record(context.Background(), float64(count))
	}
}

func IncrementBadgerOperation(operation, status string) {
	BadgerOperationsTotal.WithLabelValues(operation, status).Inc()

	if otelBadgerOperationsTotal != nil {
		otelBadgerOperationsTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		))
	}
}

func IncrementAuthAttempt(result string) {
	AuthAttemptsTotal.WithLabelValues(result).Inc()

	if otelAuthAttemptsTotal != nil {
		otelAuthAttemptsTotal.Add(context.Background(), 1, metric.WithAttributes(
			attribute.String("result", result),
		))
	}
}
