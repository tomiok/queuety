package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ServiceName = "queuety"

func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	tracer := GetTracer(ServiceName)

	if len(opts) == 0 {
		opts = []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindInternal),
		}
	}

	return tracer.Start(ctx, spanName, opts...)
}

func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	span.End()
}

func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}

	span.SetAttributes(attrs...)
}

func RecordSpanError(span trace.Span, err error, opts ...trace.EventOption) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

func WithSpanKind(kind trace.SpanKind) trace.SpanStartOption {
	return trace.WithSpanKind(kind)
}

func StringAttribute(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

func IntAttribute(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

func BoolAttribute(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}
