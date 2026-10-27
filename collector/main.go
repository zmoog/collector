// Program collector is an OpenTelemetry Collector binary.
package main

import (
	"log"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

func main() {
	if err := run(collectorSettings()); err != nil {
		log.Fatal(err)
	}
}

func collectorSettings() otelcol.CollectorSettings {
	info := component.BuildInfo{
		Command:     "collector",
		Description: "OTel Collector distribution",
		Version:     "0.5",
	}

	return otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				ProviderFactories: []confmap.ProviderFactory{
					envprovider.NewFactory(),
					fileprovider.NewFactory(),
					yamlprovider.NewFactory(),
				},
			},
		},
		ProviderModules: map[string]string{
			envprovider.NewFactory().Create(confmap.ProviderSettings{}).Scheme():  "go.opentelemetry.io/collector/confmap/provider/envprovider v1.70.0",
			fileprovider.NewFactory().Create(confmap.ProviderSettings{}).Scheme(): "go.opentelemetry.io/collector/confmap/provider/fileprovider v1.70.0",
			yamlprovider.NewFactory().Create(confmap.ProviderSettings{}).Scheme(): "go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.70.0",
		},
		ConverterModules: []string{},
	}
}

func runInteractive(params otelcol.CollectorSettings) error {
	cmd := otelcol.NewCommand(params)
	if err := cmd.Execute(); err != nil {
		log.Fatalf("collector server run finished with error: %v", err)
	}

	return nil
}
