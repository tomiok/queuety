package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Metrics struct {
	// Message metrics
	MessagesPublished metric.Int64Counter
	MessagesDelivered metric.Int64Counter
	MessagesFailed    metric.Int64Counter
	MessageDuration   metric.Float64Histogram

	// Topic metrics
	TopicsTotal      metric.Int64UpDownCounter
	SubscribersTotal metric.Int64UpDownCounter

	// Connection metrics
	ActiveConnections metric.Int64UpDownCounter
	BytesSent         metric.Int64Counter
	BytesReceived     metric.Int64Counter

	// Persistence metrics
	BadgerOps       metric.Int64Counter
	BadgerDuration  metric.Float64Histogram
	UndeliveredMsgs metric.Int64UpDownCounter

	// Authentication metrics
	AuthAttempts metric.Int64Counter
}

func newMetrics(meter metric.Meter) (*Metrics, error) {
	messagesPublished, err := meter.Int64Counter(
		"queuety_messages_published_total",
		metric.WithDescription("Total published messages"),
	)
	if err != nil {
		return nil, err
	}

	messagesDelivered, err := meter.Int64Counter(
		"queuety_messages_delivered_total",
		metric.WithDescription("Total delivered messages"),
	)
	if err != nil {
		return nil, err
	}

	messagesFailed, err := meter.Int64Counter(
		"queuety_messages_failed_total",
		metric.WithDescription("Total failed messages"),
	)
	if err != nil {
		return nil, err
	}

	messageDuration, err := meter.Float64Histogram(
		"queuety_message_processing_seconds",
		metric.WithDescription("Message processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	topicsTotal, err := meter.Int64UpDownCounter(
		"queuety_topics_total",
		metric.WithDescription("Total number of topics"),
	)
	if err != nil {
		return nil, err
	}

	subscribersTotal, err := meter.Int64UpDownCounter(
		"queuety_subscribers_total",
		metric.WithDescription("Total number of subscribers"),
	)
	if err != nil {
		return nil, err
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"queuety_active_connections",
		metric.WithDescription("Number of active TCP connections"),
	)
	if err != nil {
		return nil, err
	}

	bytesSent, err := meter.Int64Counter(
		"queuety_bytes_sent_total",
		metric.WithDescription("Total bytes sent"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	bytesReceived, err := meter.Int64Counter(
		"queuety_bytes_received_total",
		metric.WithDescription("Total bytes received"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	badgerOps, err := meter.Int64Counter(
		"queuety_badger_operations_total",
		metric.WithDescription("Total BadgerDB operations"),
	)
	if err != nil {
		return nil, err
	}

	badgerDuration, err := meter.Float64Histogram(
		"queuety_badger_operation_seconds",
		metric.WithDescription("BadgerDB operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	undeliveredMsgs, err := meter.Int64UpDownCounter(
		"queuety_undelivered_messages_total",
		metric.WithDescription("Number of undelivered messages"),
	)
	if err != nil {
		return nil, err
	}

	authAttempts, err := meter.Int64Counter(
		"queuety_auth_attempts_total",
		metric.WithDescription("Total authentication attempts"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		MessagesPublished: messagesPublished,
		MessagesDelivered: messagesDelivered,
		MessagesFailed:    messagesFailed,
		MessageDuration:   messageDuration,
		TopicsTotal:       topicsTotal,
		SubscribersTotal:  subscribersTotal,
		ActiveConnections: activeConnections,
		BytesSent:         bytesSent,
		BytesReceived:     bytesReceived,
		BadgerOps:         badgerOps,
		BadgerDuration:    badgerDuration,
		UndeliveredMsgs:   undeliveredMsgs,
		AuthAttempts:      authAttempts,
	}, nil
}

// Helper methods for easy metric recording

// Message operations
func (t *Telemetry) IncrementMessagesPublished(ctx context.Context, topic string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.MessagesPublished.Add(ctx, 1, metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

func (t *Telemetry) IncrementMessagesDelivered(ctx context.Context, topic string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.MessagesDelivered.Add(ctx, 1, metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

func (t *Telemetry) IncrementMessagesFailed(ctx context.Context, topic, reason string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.MessagesFailed.Add(ctx, 1, metric.WithAttributes(
		attribute.String("topic", topic),
		attribute.String("reason", reason),
	))
}

func (t *Telemetry) RecordMessageDuration(ctx context.Context, duration float64, topic, operation string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.MessageDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("topic", topic),
		attribute.String("operation", operation),
	))
}

// Topic operations
func (t *Telemetry) IncrementTopics(ctx context.Context) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.TopicsTotal.Add(ctx, 1)
}

func (t *Telemetry) DecrementTopics(ctx context.Context) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.TopicsTotal.Add(ctx, -1)
}

func (t *Telemetry) IncrementSubscribers(ctx context.Context, topic string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.SubscribersTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

func (t *Telemetry) DecrementSubscribers(ctx context.Context, topic string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.SubscribersTotal.Add(ctx, -1, metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

// Connection metrics
func (t *Telemetry) IncrementActiveConnections(ctx context.Context) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.ActiveConnections.Add(ctx, 1)
}

func (t *Telemetry) DecrementActiveConnections(ctx context.Context) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.ActiveConnections.Add(ctx, -1)
}

func (t *Telemetry) IncrementBytesSent(ctx context.Context, bytes int64) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.BytesSent.Add(ctx, bytes)
}

func (t *Telemetry) IncrementBytesReceived(ctx context.Context, bytes int64) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.BytesReceived.Add(ctx, bytes)
}

// BadgerDB metrics
func (t *Telemetry) IncrementBadgerOps(ctx context.Context, operation, status string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.BadgerOps.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("status", status),
	))
}

func (t *Telemetry) RecordBadgerDuration(ctx context.Context, duration float64, operation string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.BadgerDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("operation", operation),
	))
}

func (t *Telemetry) UpdateUndeliveredMessages(ctx context.Context, count int64) {
	if !t.enabled || t.metrics == nil {
		return
	}
	// Reset and set the current count
	t.metrics.UndeliveredMsgs.Add(ctx, count)
}

// Authentication metrics
func (t *Telemetry) IncrementAuthAttempts(ctx context.Context, result string) {
	if !t.enabled || t.metrics == nil {
		return
	}
	t.metrics.AuthAttempts.Add(ctx, 1, metric.WithAttributes(
		attribute.String("result", result),
	))
}

// Tracing helper methods
func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name)
}

func (t *Telemetry) StartMessagePublishSpan(ctx context.Context, topic, messageID string) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, "message.publish")
	span.SetAttributes(
		attribute.String("topic", topic),
		attribute.String("message_id", messageID),
		attribute.String("operation", "publish"),
	)
	return ctx, span
}

func (t *Telemetry) StartMessageDeliverSpan(ctx context.Context, topic, messageID string) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, "message.deliver")
	span.SetAttributes(
		attribute.String("topic", topic),
		attribute.String("message_id", messageID),
		attribute.String("operation", "deliver"),
	)
	return ctx, span
}

func (t *Telemetry) StartBadgerSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, "badger."+operation)
	span.SetAttributes(
		attribute.String("db.system", "badgerdb"),
		attribute.String("db.operation", operation),
	)
	return ctx, span
}

// Utility function to measure duration
func (t *Telemetry) MeasureDuration(start time.Time, ctx context.Context, recordFunc func(context.Context, float64)) {
	if !t.enabled {
		return
	}
	duration := time.Since(start).Seconds()
	recordFunc(ctx, duration)
}
