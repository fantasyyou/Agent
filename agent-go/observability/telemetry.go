package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type requestKey struct{}

func Init(ctx context.Context) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "agent-go")))}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithTimeout(3*time.Second))
		if err != nil {
			return nil, err
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func WithRequest(ctx context.Context) context.Context {
	if RequestID(ctx) != "" {
		return ctx
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return context.WithValue(ctx, requestKey{}, "req_"+hex.EncodeToString(bytes))
}
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestKey{}).(string)
	return value
}
func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("agent-go").Start(ctx, name)
	span.SetAttributes(attribute.String("request_id", RequestID(ctx)))
	return ctx, span
}
func Stage(ctx context.Context, name string) (context.Context, func(error)) {
	ctx, span := Start(ctx, name)
	started := time.Now()
	return ctx, func(err error) {
		result := "success"
		if err != nil {
			result = "failed"
			span.SetStatus(codes.Error, name+" failed")
			span.SetAttributes(attribute.String("error.type", name+"_failed"))
		}
		slog.InfoContext(ctx, "stage_completed", "stage", name, "result", result, "duration_ms", time.Since(started).Milliseconds())
		span.End()
	}
}

type Handler struct{ slog.Handler }

func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	record.AddAttrs(slog.String("service", "agent-go"))
	if id := RequestID(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	if sc.IsValid() {
		record.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}
func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{h.Handler.WithAttrs(attrs)}
}
func (h Handler) WithGroup(name string) slog.Handler { return Handler{h.Handler.WithGroup(name)} }
