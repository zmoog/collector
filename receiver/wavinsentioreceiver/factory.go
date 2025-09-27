package wavinsentioreceiver

import (
	"context"
	"fmt"
	"time"

	"github.com/zmoog/collector/receiver/wavinsentioreceiver/internal/metadata"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	DefaultCollectionInterval = 60 * time.Second
)

var (
	typeStr = component.MustNewType("wavinsentio")
)

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = DefaultCollectionInterval
	return &Config{
		ControllerConfig: cfg,
	}
}

func createMetricsReceiver(ctx context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	// logger := settings.Logger
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	var metrics scraper.Metrics

	wavinsentioScraper := newScraper(cfg, settings)

	metrics, err := scraper.NewMetrics(
		wavinsentioScraper.scrape,
		scraper.WithStart(wavinsentioScraper.start),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		settings,
		consumer,
		scraperhelper.AddScraper(metadata.Type, metrics),
	)

	// rcvr := wavinsentioReceiver{
	// 	logger:              logger,
	// 	consumer:            consumer,
	// 	config:              &config,
	// 	scraper:             scraper,
	// 	locationUnmarshaler: &locationUnmarshaler{logger: logger},
	// 	roomUnmarshaler:     &roomUnmarshaler{logger: logger},
	// }

	// return &rcvr, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
	)
}
