package shellycloudreceiver

import (
	"context"
	"fmt"
	"time"

	"github.com/zmoog/collector/receiver/shellycloudreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

const (
	DefaultCollectionInterval = 60 * time.Second
)

var typeStr = component.MustNewType("shellycloud")

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = DefaultCollectionInterval
	return &Config{
		ControllerConfig: cfg,
	}
}

func createMetricsReceiver(_ context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	s := newScraper(cfg, settings)

	sc, err := scraper.NewMetrics(
		s.scrape,
		scraper.WithStart(s.start),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		settings,
		consumer,
		scraperhelper.AddScraper(metadata.Type, sc),
	)
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
	)
}
