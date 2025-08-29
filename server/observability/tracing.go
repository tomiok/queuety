package observability

import (
	"context"
	"fmt"

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

type SpanHelper struct {
	span trace.Span
}

func NewSpanHelper(ctx context.Context, name string, kind trace.SpanKind) *SpanHelper {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	return &SpanHelper{span: span}
}

func NewSpanHelperWithContext(ctx context.Context, name string, kind trace.SpanKind) *SpanHelper {
	if ctx == nil {
		ctx = context.Background()
	}
	return NewSpanHelper(ctx, name, kind)
}

func (h *SpanHelper) End(err error) {
	EndSpan(h.span, err)
}

func (h *SpanHelper) AddAttributes(attrs ...attribute.KeyValue) {
	AddSpanAttributes(h.span, attrs...)
}

func (h *SpanHelper) SetError(err error) {
	if err != nil {
		RecordSpanError(h.span, err)
	}
}

func (h *SpanHelper) WithRecover(fn func() error) {
	defer func() {
		if r := recover(); r != nil {
			EndSpan(h.span, fmt.Errorf("panic: %v", r))
		}
	}()

	if err := fn(); err != nil {
		h.End(err)
	} else {
		h.End(nil)
	}
}

func (h *SpanHelper) WithRecoverNoReturn(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			EndSpan(h.span, fmt.Errorf("panic: %v", r))
		}
	}()

	fn()
	h.End(nil)
}

func (h *SpanHelper) GetSpan() trace.Span {
	return h.span
}

func StartSpanSafe(ctx context.Context, name string, kind trace.SpanKind) func() {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	return func() {
		EndSpan(span, nil)
	}
}

func StartSpanSafeWithError(ctx context.Context, name string, kind trace.SpanKind) func(*error) {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	return func(err *error) {
		if *err != nil {
			RecordSpanError(span, *err)
		}
		EndSpan(span, *err)
	}
}

func StartSpanSafeWithRecover(ctx context.Context, name string, kind trace.SpanKind) func() {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	return func() {
		defer func() {
			if r := recover(); r != nil {
				EndSpan(span, fmt.Errorf("panic: %v", r))
			}
		}()
		EndSpan(span, nil)
	}
}

func StartSpanWithAttributes(ctx context.Context, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) func() {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	AddSpanAttributes(span, attrs...)
	return func() {
		EndSpan(span, nil)
	}
}

func StartSpanWithAttributesAndError(ctx context.Context, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) func(*error) {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	AddSpanAttributes(span, attrs...)
	return func(err *error) {
		if *err != nil {
			RecordSpanError(span, *err)
		}
		EndSpan(span, *err)
	}
}

func SpanWithAttributes(ctx context.Context, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) (trace.Span, func()) {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	AddSpanAttributes(span, attrs...)
	return span, func() {
		EndSpan(span, nil)
	}
}

func SpanWithAttributesAndError(ctx context.Context, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) (trace.Span, func(*error)) {
	_, span := StartSpan(ctx, name, WithSpanKind(kind))
	AddSpanAttributes(span, attrs...)
	return span, func(err *error) {
		if *err != nil {
			RecordSpanError(span, *err)
		}
		EndSpan(span, *err)
	}
}
