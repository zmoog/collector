package zcsazzurroreceiver

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-freelru"
	"github.com/zmoog/collector/receiver/zcsazzurroreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

var (
	typeStr = component.MustNewType("zcsazzurro")
)

const (
	DefaultCollectionInterval = 5 * time.Minute
)

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = DefaultCollectionInterval
	return &Config{
		ControllerConfig: cfg,
	}
}

// hashString from https://github.com/elastic/go-freelru/blob/237b2bf67a116266a3660add83b1809373dc0ac7/shardedlru_test.go#L96C1-L103C2
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

func createMetricsReceiver(ctx context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	// Initialize LRU cache with capacity for 1000 things
	cache, err := freelru.NewSynced[string, time.Time](1000, hashString)
	if err != nil {
		return nil, err
	}

	zcsazzurroScraper := newScraper(cfg, settings, cache)

	metrics, err := scraper.NewMetrics(
		zcsazzurroScraper.scrape,
		scraper.WithStart(zcsazzurroScraper.start),
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
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
	)
}
