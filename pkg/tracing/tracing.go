package tracing

// TODO: Deprecated: This module is no longer supported. OpenTelemetry dropped support for Jaeger exporter in July 2023. Jaeger officially accepts and recommends using OTLP. Use go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp or go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc instead.

import (
	"context"
	"io"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Tracer provides distributed tracing functionality
type Tracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
	logger   logger.Logger
	mu       sync.RWMutex
}

// Config holds configuration for tracing
type Config struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
	Enabled        bool
}

// NewTracer creates a new tracer
func NewTracer(config Config, logger logger.Logger) (*Tracer, error) {
	if !config.Enabled {
		// Return a no-op tracer if tracing is disabled
		return &Tracer{
			provider: trace.NewNoopTracerProvider(),
			tracer:   trace.NewNoopTracerProvider().Tracer(""),
			logger:   logger.With("component", "tracer"),
		}, nil
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Endpoint)))
	if err != nil {
		return nil, err
	}

	// Create trace provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("environment", "production"),
		)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer := tp.Tracer(config.ServiceName)

	return &Tracer{
		provider: tp,
		tracer:   tracer,
		logger:   logger.With("component", "tracer"),
	}, nil
}

// Shutdown shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Shutdown trace provider
	if provider, ok := t.provider.(io.Closer); ok {
		return provider.Close()
	}

	return nil
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from the context
func (t *Tracer) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func (t *Tracer) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func (t *Tracer) SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func (t *Tracer) RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
}

// End ends the current span
func (t *Tracer) End(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.End()
}
