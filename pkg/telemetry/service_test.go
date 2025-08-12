package telemetry

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	service := NewService(config)

	if service == nil {
		t.Fatal("Expected service to be created")
	}
	if service.config != config {
		t.Error("Expected service config to match input config")
	}
	if service.isInitialized {
		t.Error("Expected service to not be initialized initially")
	}
}

func TestServiceInitializeWithoutTelemetry(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        false,
		OTLPEndpoint:   "",
	}

	service := NewService(config)
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Expected no error during initialization, got %v", err)
	}
	if !service.isInitialized {
		t.Error("Expected service to be marked as initialized")
	}
}

func TestServiceInitializeWithOTLP(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
		OTLPEndpoint:   "localhost:4317",
		DeviceID:       "test-device-id",
	}

	service := NewService(config)
	ctx := context.Background()

	// This will fail because there's no actual OTLP server, but we test the initialization logic
	err := service.Initialize(ctx)
	// We expect this to succeed even if OTLP fails, as it should continue without telemetry
	if err != nil {
		t.Errorf("Expected initialization to succeed even with OTLP failure, got %v", err)
	}
}

func TestServiceTrackToolInvocationNotInitialized(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	service := NewService(config)
	ctx := context.Background()

	// Should not panic or error when not initialized
	service.TrackToolInvocation(ctx, "kubectl", "get", true)
	service.TrackServiceStartup(ctx)
}

func TestServiceTrackingAfterInitialization(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
		OTLPEndpoint:   "", // No OTLP to avoid connection issues in tests
		DeviceID:       "test-device-id",
	}

	service := NewService(config)
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// These should not panic after initialization
	service.TrackToolInvocation(ctx, "kubectl", "get", true)
	service.TrackServiceStartup(ctx)
}

func TestServiceStartActivity(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
		OTLPEndpoint:   "",
	}

	service := NewService(config)
	ctx := context.Background()

	// Before initialization - should return context and span without error
	newCtx, span := service.StartActivity(ctx, "test-activity")
	if newCtx == nil || span == nil {
		t.Error("Expected context and span to be returned even when not initialized")
	}

	// After initialization
	err := service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	newCtx2, span2 := service.StartActivity(ctx, "test-activity-2")
	if newCtx2 == nil || span2 == nil {
		t.Error("Expected context and span to be returned after initialization")
	}
}

func TestServiceShutdown(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
		OTLPEndpoint:   "",
	}

	service := NewService(config)
	ctx := context.Background()

	// Shutdown before initialization should not error
	err := service.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error during shutdown of uninitialized service, got %v", err)
	}

	// Initialize and then shutdown
	err = service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	err = service.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error during shutdown, got %v", err)
	}

	if service.isInitialized {
		t.Error("Expected service to be marked as not initialized after shutdown")
	}
}

func TestServiceDisabledTelemetry(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        false,
	}

	service := NewService(config)
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Expected no error during initialization with disabled telemetry, got %v", err)
	}

	// All tracking methods should work without error
	service.TrackToolInvocation(ctx, "kubectl", "get", true)
	service.TrackServiceStartup(ctx)

	newCtx, span := service.StartActivity(ctx, "test-activity")
	if newCtx == nil || span == nil {
		t.Error("Expected context and span to be returned even with disabled telemetry")
	}

	err = service.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error during shutdown with disabled telemetry, got %v", err)
	}
}
