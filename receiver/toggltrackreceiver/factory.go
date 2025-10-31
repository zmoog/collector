package toggltrackreceiver

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/zmoog/collector/receiver/toggltrackreceiver/internal/metadata"
)

var (
	typeStr = component.MustNewType("toggltrack")
)

const (
	DefaultCollectionInterval = 1 * time.Minute
	DefaultLookback           = 24 * 30 * time.Hour // 30 days
)

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = DefaultCollectionInterval
	return &Config{
		ControllerConfig: cfg,
		Lookback:         DefaultLookback.String(),
	}
}

// createScraperFactory creates a scraper.Factory for toggltrack logs
func createScraperFactory(cfg *Config, settings receiver.Settings) scraper.Factory {
	return scraper.NewFactory(
		metadata.Type,
		func() component.Config { return cfg },
		scraper.WithLogs(func(ctx context.Context, scraperSettings scraper.Settings, scraperCfg component.Config) (scraper.Logs, error) {
			cfg, ok := scraperCfg.(*Config)
			if !ok {
				return nil, fmt.Errorf("invalid config type")
			}
			togglTrackScraper := newScraper(cfg, settings)
			return scraper.NewLogs(
				togglTrackScraper.scrape,
				scraper.WithStart(togglTrackScraper.start),
			)
		}, component.StabilityLevelAlpha),
	)
}

func createLogsReceiver(ctx context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	scraperFactory := createScraperFactory(cfg, settings)

	return scraperhelper.NewLogsController(
		&cfg.ControllerConfig,
		settings,
		consumer,
		scraperhelper.AddFactoryWithConfig(scraperFactory, cfg),
	)
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha),
	)
}
