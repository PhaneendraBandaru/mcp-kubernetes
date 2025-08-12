package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TelemetryInterface defines the methods available for telemetry tracking
type TelemetryInterface interface {
	// Initialize sets up the telemetry providers and exporters
	Initialize(ctx context.Context) error

	// Shutdown gracefully shuts down the telemetry service
	Shutdown(ctx context.Context) error

	// StartActivity starts a new telemetry activity (span)
	StartActivity(ctx context.Context, activityName string) (context.Context, trace.Span)

	// TrackToolInvocation tracks a tool invocation with minimal data
	TrackToolInvocation(ctx context.Context, toolName string, operation string, success bool)

	// TrackServiceStartup tracks the MCP server startup
	TrackServiceStartup(ctx context.Context)
}
