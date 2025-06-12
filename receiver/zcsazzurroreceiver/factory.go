package zcsazzurroreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("zcsazzurro")
)

const (
	defaultInterval = 5 * time.Minute
)

func createDefaultConfig() component.Config {
	return Config{
		Interval: defaultInterval.String(),
	}
}

func createMetricsReceiver(ctx context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	logger := settings.Logger
	config := baseCfg.(Config)
	scraper := NewScraper(config.ClientID, config.AuthKey, config.ThingKey, logger)

	rcvr := zcsazzurroReceiver{
		logger:    logger,
		consumer:  consumer,
		config:    &config,
		scraper:   scraper,
		marshaler: newAzzurroRealtimeDataMarshaler(logger),
	}

	return &rcvr, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
	)
}
