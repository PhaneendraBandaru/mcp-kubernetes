package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Service provides telemetry functionality for Kubernetes MCP
type Service struct {
	config            *Config
	tracer            oteltrace.Tracer
	tracerProvider    *trace.TracerProvider
	appInsightsClient appinsights.TelemetryClient
	isInitialized     bool
}

// NewService creates a new telemetry service
func NewService(config *Config) *Service {
	return &Service{
		config:        config,
		isInitialized: false,
	}
}

// Initialize sets up the telemetry providers and exporters
func (s *Service) Initialize(ctx context.Context) error {
	// Initialize tracers and exporters
	if err := s.initializeTracing(ctx); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize Application Insights if configured
	if s.config.HasApplicationInsights() {
		s.initializeApplicationInsights()
	}

	s.isInitialized = true
	return nil
}

// initializeTracing sets up OpenTelemetry tracing
func (s *Service) initializeTracing(ctx context.Context) error {
	if !s.config.HasOTLP() {
		return nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", s.config.ServiceName),
			attribute.String("service.version", s.config.ServiceVersion),
			attribute.String("device.id", s.config.DeviceID),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Add OTLP exporter
	var exporters []trace.SpanExporter
	otlpExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(s.config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Failed to create OTLP gRPC exporter: %v", err)
	} else {
		exporters = append(exporters, otlpExporter)
	}

	// Create tracer provider with batch span processor
	var options []trace.TracerProviderOption
	options = append(options, trace.WithResource(res))
	for _, exporter := range exporters {
		processor := trace.NewBatchSpanProcessor(exporter)
		options = append(options, trace.WithSpanProcessor(processor))
	}

	// Add sampler
	options = append(options, trace.WithSampler(trace.AlwaysSample()))
	s.tracerProvider = trace.NewTracerProvider(options...)

	// Set global tracer provider
	otel.SetTracerProvider(s.tracerProvider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	s.tracer = otel.Tracer(s.config.ServiceName)
	return nil
}

// initializeApplicationInsights sets up Application Insights client
func (s *Service) initializeApplicationInsights() {
	if !s.config.Enabled {
		return
	}

	// Create TelemetryConfiguration
	config := appinsights.NewTelemetryConfiguration(s.config.instrumentationKey)
	s.appInsightsClient = appinsights.NewTelemetryClientFromConfig(config)

	// Add common properties
	commonProps := s.appInsightsClient.Context().CommonProperties
	commonProps["service.name"] = s.config.ServiceName
	commonProps["service.version"] = s.config.ServiceVersion
	commonProps["device.id"] = s.config.DeviceID
}

// StartActivity starts a new telemetry activity (span)
func (s *Service) StartActivity(ctx context.Context, activityName string) (context.Context, oteltrace.Span) {
	if !s.isInitialized || s.tracer == nil {
		// Return a no-op span if telemetry is not initialized
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	return s.tracer.Start(ctx, activityName)
}

// TrackToolInvocation tracks a tool invocation with minimal data
func (s *Service) TrackToolInvocation(ctx context.Context, toolName string, operation string, success bool) {
	if !s.isInitialized {
		return
	}

	// Send to OTLP as a span if available
	if s.config.HasOTLP() && s.tracer != nil {
		_, span := s.tracer.Start(ctx, "ToolInvocation")
		defer span.End()

		span.SetAttributes(
			attribute.String("tool.name", toolName),
			attribute.String("tool.operation", operation),
			attribute.Bool("tool.success", success),
		)
	}

	// Send to Application Insights as a trace
	if s.config.HasApplicationInsights() && s.appInsightsClient != nil {
		event := appinsights.NewTraceTelemetry("ToolInvocation", appinsights.Information)
		event.Properties["tool.name"] = toolName
		event.Properties["tool.operation"] = operation
		event.Properties["tool.success"] = fmt.Sprintf("%v", success)
		s.appInsightsClient.Track(event)
	}
}

// TrackServiceStartup tracks the MCP server startup
func (s *Service) TrackServiceStartup(ctx context.Context) {
	if !s.isInitialized {
		return
	}

	// Send to OTLP as a span if available
	if s.config.HasOTLP() && s.tracer != nil {
		_, span := s.tracer.Start(ctx, "ServiceStartup")
		defer span.End()

		span.SetAttributes(
			attribute.String("service.event", "startup"),
			attribute.String("service.name", s.config.ServiceName),
			attribute.String("service.version", s.config.ServiceVersion),
		)
	}

	// Send to Application Insights as an event
	if s.config.HasApplicationInsights() && s.appInsightsClient != nil {
		event := appinsights.NewTraceTelemetry("ServiceStartup", appinsights.Information)
		event.Properties["service.name"] = s.config.ServiceName
		event.Properties["service.version"] = s.config.ServiceVersion
		s.appInsightsClient.Track(event)
	}
}

// Shutdown gracefully shuts down the telemetry service
func (s *Service) Shutdown(ctx context.Context) error {
	if !s.isInitialized {
		return nil
	}

	// Flush Application Insights if configured
	if s.config.HasApplicationInsights() && s.appInsightsClient != nil {
		<-s.appInsightsClient.Channel().Close(5 * time.Second)
	}

	// Shutdown tracer provider
	if s.tracerProvider != nil {
		if err := s.tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	s.isInitialized = false
	return nil
}
