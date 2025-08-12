package telemetry

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		serviceVersion  string
		envVars         map[string]string
		expectedEnabled bool
	}{
		{
			name:            "Default enabled",
			serviceName:     "test-service",
			serviceVersion:  "1.0.0",
			envVars:         map[string]string{},
			expectedEnabled: true,
		},
		{
			name:           "Disabled via env var",
			serviceName:    "test-service",
			serviceVersion: "1.0.0",
			envVars: map[string]string{
				"KUBERNETES_MCP_COLLECT_TELEMETRY": "false",
			},
			expectedEnabled: false,
		},
		{
			name:           "Enabled via env var",
			serviceName:    "test-service",
			serviceVersion: "1.0.0",
			envVars: map[string]string{
				"KUBERNETES_MCP_COLLECT_TELEMETRY": "true",
			},
			expectedEnabled: true,
		},
		{
			name:           "Invalid env var defaults to enabled",
			serviceName:    "test-service",
			serviceVersion: "1.0.0",
			envVars: map[string]string{
				"KUBERNETES_MCP_COLLECT_TELEMETRY": "invalid",
			},
			expectedEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			config := NewConfig(tt.serviceName, tt.serviceVersion)

			if config.ServiceName != tt.serviceName {
				t.Errorf("Expected ServiceName %s, got %s", tt.serviceName, config.ServiceName)
			}
			if config.ServiceVersion != tt.serviceVersion {
				t.Errorf("Expected ServiceVersion %s, got %s", tt.serviceVersion, config.ServiceVersion)
			}
			if config.Enabled != tt.expectedEnabled {
				t.Errorf("Expected Enabled %v, got %v", tt.expectedEnabled, config.Enabled)
			}

			// When enabled, DeviceID should be generated
			if tt.expectedEnabled && config.DeviceID == "" {
				t.Error("Expected DeviceID to be generated when telemetry is enabled")
			}
		})
	}
}

func TestConfigHasOTLP(t *testing.T) {
	config := &Config{
		OTLPEndpoint: "",
	}
	if config.HasOTLP() {
		t.Error("Expected HasOTLP to return false when endpoint is empty")
	}

	config.OTLPEndpoint = "localhost:4317"
	if !config.HasOTLP() {
		t.Error("Expected HasOTLP to return true when endpoint is set")
	}
}

func TestConfigHasApplicationInsights(t *testing.T) {
	tests := []struct {
		name               string
		enabled            bool
		instrumentationKey string
		expected           bool
	}{
		{
			name:               "Disabled telemetry",
			enabled:            false,
			instrumentationKey: "test-key",
			expected:           false,
		},
		{
			name:               "Enabled with key",
			enabled:            true,
			instrumentationKey: "test-key",
			expected:           true,
		},
		{
			name:               "Enabled without key",
			enabled:            true,
			instrumentationKey: "",
			expected:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Enabled:            tt.enabled,
				instrumentationKey: tt.instrumentationKey,
			}

			result := config.HasApplicationInsights()
			if result != tt.expected {
				t.Errorf("Expected HasApplicationInsights %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfigSetOTLPEndpoint(t *testing.T) {
	config := &Config{}
	endpoint := "localhost:4317"

	config.SetOTLPEndpoint(endpoint)

	if config.OTLPEndpoint != endpoint {
		t.Errorf("Expected OTLPEndpoint %s, got %s", endpoint, config.OTLPEndpoint)
	}
}

func TestGenerateDeviceID(t *testing.T) {
	deviceID := generateDeviceID()

	if deviceID == "" {
		t.Error("Expected device ID to be generated")
	}

	// Device ID should be consistent
	deviceID2 := generateDeviceID()
	if deviceID != deviceID2 {
		t.Error("Expected device ID to be consistent across calls")
	}
}

func TestGetApplicationInsightsInstrumentationKey(t *testing.T) {
	// Test with custom key
	customKey := "custom-instrumentation-key"
	_ = os.Setenv("APPLICATIONINSIGHTS_INSTRUMENTATION_KEY", customKey)
	defer func() { _ = os.Unsetenv("APPLICATIONINSIGHTS_INSTRUMENTATION_KEY") }()

	key := getApplicationInsightsInstrumentationKey()
	if key != customKey {
		t.Errorf("Expected custom key %s, got %s", customKey, key)
	}

	// Test with default key
	_ = os.Unsetenv("APPLICATIONINSIGHTS_INSTRUMENTATION_KEY")
	key = getApplicationInsightsInstrumentationKey()
	if key != defaultInstrumentationKey {
		t.Errorf("Expected default key %s, got %s", defaultInstrumentationKey, key)
	}
}
