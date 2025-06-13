package zcsazzurroreceiver

import (
	"context"
	"hash/maphash"
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

// hashString provides a hash function for string keys
func hashString(s string) uint32 {
	h := maphash.Hash{}
	h.WriteString(s)
	return uint32(h.Sum64())
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
