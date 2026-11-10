package main

import (
	"context"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
)

func TestComponents(t *testing.T) {
	factories, err := components()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"shellycloud", "toggltrack", "wavinsentio", "zcsazzurro"} {
		if factories.Receivers[component.MustNewType(name)] == nil {
			t.Errorf("receiver %s is not registered", name)
		}
	}
	for _, name := range []string{"debug", "otlp", "elasticsearch"} {
		if factories.Exporters[component.MustNewType(name)] == nil {
			t.Errorf("exporter %s is not registered", name)
		}
	}
	for _, name := range []string{"basicauth", "health_check"} {
		if factories.Extensions[component.MustNewType(name)] == nil {
			t.Errorf("extension %s is not registered", name)
		}
	}
	if factories.Processors[component.MustNewType("batch")] == nil || factories.Telemetry == nil {
		t.Error("batch processor or telemetry factory is missing")
	}
}

// Exercise the actual binary's providers and the upstream config decoder and
// validators without contacting any third-party services.
func TestCollectorLifecycle(t *testing.T) {
	settings := collectorSettings()
	settings.DisableGracefulShutdown = true
	settings.ConfigProviderSettings.ResolverSettings.URIs = []string{
		"testdata/config.yaml",
		// Keep the real receiver and batch pipelines, but avoid cloud requests,
		// external exporters, and fixed listening ports during this smoke test.
		"yaml:" + `
receivers:
  toggltrack:
    initial_delay: 1h
  wavinsentio:
    initial_delay: 1h
  zcsazzurro/thing1:
    initial_delay: 1h
  zcsazzurro/thing2:
    initial_delay: 1h
  shellycloud:
    initial_delay: 1h
extensions:
  health_check:
    endpoint: localhost:0
service:
  pipelines:
    logs:
      exporters: [debug]
    metrics:
      exporters: [debug]
  telemetry:
    metrics:
      level: none
`,
	}
	collector, err := otelcol.NewCollector(settings)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- collector.Run(ctx) }()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for collector.GetState() != otelcol.StateRunning {
		select {
		case err := <-result:
			t.Fatalf("collector exited before becoming ready: %v", err)
		case <-ctx.Done():
			t.Fatal("collector did not become ready")
		case <-ticker.C:
		}
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("collector did not shut down")
	}
	if collector.GetState() != otelcol.StateClosed {
		t.Errorf("unexpected state after shutdown: %v", collector.GetState())
	}
}

func TestConfigCompatibility(t *testing.T) {
	for key, value := range map[string]string{
		"TOGGL_API_TOKEN":         "test-token",
		"WS_USERNAME":             "test-user",
		"WS_PASSWORD":             "test-password",
		"ZCS_CLIENT_ID":           "test-client",
		"ZCS_AUTH_KEY":            "test-key",
		"ZCS_THING_KEY":           "test-thing",
		"ELASTICSEARCH_ENDPOINTS": "https://localhost:9200",
		"ELASTICSEARCH_API_KEY":   "test-key",
	} {
		t.Setenv(key, value)
	}
	for _, key := range []string{"TOGGL_INTERVAL", "WS_INTERVAL", "WS_ENDPOINT", "WS_WEB_API_KEY"} {
		// The env provider uses defaults only for unset variables, not empty ones.
		// Setenv registers cleanup to restore the caller's environment.
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	for _, uri := range []string{"../config.yaml", "testdata/config.yaml"} {
		t.Run(uri, func(t *testing.T) {
			settings := collectorSettings()
			settings.ConfigProviderSettings.ResolverSettings.URIs = []string{uri}
			provider, err := otelcol.NewConfigProvider(settings.ConfigProviderSettings)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Error(err)
				}
			})
			factories, err := settings.Factories()
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := provider.Get(context.Background(), factories)
			if err != nil {
				t.Fatal(err)
			}
			if err := confmap.Validate(cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}
