package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/zmoog/collector/receiver/toggltrackreceiver"
	"github.com/zmoog/collector/receiver/wavinsentioreceiver"
	"github.com/zmoog/collector/receiver/zcsazzurroreceiver"

	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

func components() (otelcol.Factories, error) {
	var err error
	factories := otelcol.Factories{}

	factories.Extensions, err = otelcol.MakeFactoryMap[extension.Factory](
		basicauthextension.NewFactory(),
		healthcheckextension.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Receivers, err = otelcol.MakeFactoryMap[receiver.Factory](
		toggltrackreceiver.NewFactory(),
		zcsazzurroreceiver.NewFactory(),
		wavinsentioreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Exporters, err = otelcol.MakeFactoryMap[exporter.Factory](
		debugexporter.NewFactory(),
		otlpexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Processors, err = otelcol.MakeFactoryMap[processor.Factory](
		batchprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Connectors, err = otelcol.MakeFactoryMap[connector.Factory]()
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Telemetry = otelconftelemetry.NewFactory()

	return factories, nil
}
