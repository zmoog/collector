package zcsazzurroreceiver

import (
	"context"
	"time"

	"github.com/elastic/go-freelru"
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

// hashString from https://github.com/elastic/go-freelru/blob/237b2bf67a116266a3660add83b1809373dc0ac7/shardedlru_test.go#L96C1-L103C2
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

func createMetricsReceiver(ctx context.Context, settings receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	logger := settings.Logger
	config := baseCfg.(Config)
	scraper := NewScraper(config.ClientID, config.AuthKey, config.ThingKey, logger)

	// Initialize LRU cache with capacity for 1000 things
	cache, err := freelru.NewSynced[string, time.Time](1000, hashString)
	if err != nil {
		return nil, err
	}

	rcvr := zcsazzurroReceiver{
		logger:    logger,
		consumer:  consumer,
		config:    &config,
		scraper:   scraper,
		marshaler: newAzzurroRealtimeDataMarshaler(logger),
		cache:     cache,
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
