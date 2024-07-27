package helpers

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// GetTracerFromContext retrieves the tracer from the given context.
// If the tracer is not found in the context, a new tracer is created using the OpenTelemetry library.
// The tracer is then returned.
func GetTracerFromContext(ctx context.Context) trace.Tracer {
	tracer, ok := ctx.Value("tracer").(trace.Tracer)
	if !ok {
		tracer = otel.Tracer("defaultTracer")
	}
	return tracer
}

func StartSpanOnTracerFromContext(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := GetTracerFromContext(ctx)
	return tracer.Start(ctx, name)
}
