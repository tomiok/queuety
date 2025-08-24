package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan inicia un nuevo span con un nombre y opciones específicas
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Si no se proporciona un contexto, usar un contexto de fondo
	if ctx == nil {
		ctx = context.Background()
	}

	// Obtener el tracer por defecto
	tracer := GetTracer("queuety")

	// Iniciar el span con opciones predeterminadas si no se proporcionan
	if len(opts) == 0 {
		opts = []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindInternal),
		}
	}

	// Iniciar y devolver el span
	return tracer.Start(ctx, spanName, opts...)
}

// EndSpan termina un span, con opción de manejar errores
func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}

	// Si hay un error, marcar el span como fallido
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	// Terminar el span
	span.End()
}

// AddSpanAttributes añade atributos a un span existente
func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}

	span.SetAttributes(attrs...)
}

// RecordSpanError registra un error en un span
func RecordSpanError(span trace.Span, err error, opts ...trace.EventOption) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// WithSpanKind establece el tipo de span
func WithSpanKind(kind trace.SpanKind) trace.SpanStartOption {
	return trace.WithSpanKind(kind)
}

// Funciones de ayuda para crear atributos comunes
func StringAttribute(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

func IntAttribute(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

func BoolAttribute(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// // Ejemplo de uso de las funciones de traza
// func ExampleTraceUsage(ctx context.Context) error {
// 	// Iniciar un span
// 	ctx, span := StartSpan(
// 		ctx,
// 		"ejemplo-operacion",
// 		WithSpanKind(trace.SpanKindServer),
// 	)
// 	defer EndSpan(span, nil)

// 	// Añadir atributos
// 	AddSpanAttributes(span,
// 		StringAttribute("usuario", "john_doe"),
// 		IntAttribute("user_id", 123),
// 	)

// 	// Simular una operación que podría fallar
// 	err := realizarOperacion()
// 	if err != nil {
// 		// Registrar error en el span
// 		RecordSpanError(span, err)
// 		return err
// 	}

// 	return nil
// }

// // Función de ejemplo para demostrar manejo de errores
// func realizarOperacion() error {
// 	// Lógica de operación simulada
// 	return nil
// }
