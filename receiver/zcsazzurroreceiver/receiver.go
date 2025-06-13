package zcsazzurroreceiver

import (
	"context"
	"time"

	"github.com/elastic/go-freelru"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type zcsazzurroReceiver struct {
	cancel    context.CancelFunc
	logger    *zap.Logger
	consumer  consumer.Metrics
	config    *Config
	scraper   *Scraper
	marshaler *azzurroRealtimeDataMarshaler
	cache     *freelru.SyncedLRU[string, time.Time]
}

// shouldProcessThing checks if we should process metrics for this thing
// Returns true if:
// - We haven't seen this thing before, OR
// - The metrics timestamp is newer than what we last processed
func (z *zcsazzurroReceiver) shouldProcessThing(thingKey string, metricsTime time.Time) bool {
	z.logger.Debug("Checking if should process thing",
		zap.String("thingKey", thingKey),
		zap.Time("metricsTime", metricsTime))

	z.logger.Debug("Cache keys", zap.Any("keys", z.cache.Keys()))
	lastUpdate, exists := z.cache.Get(thingKey)
	if !exists {
		z.logger.Debug("Thing not seen before, processing", zap.String("thingKey", thingKey))
		return true
	}

	z.logger.Debug("Thing seen before, checking last update",
		zap.String("thingKey", thingKey),
		zap.Time("lastUpdate", lastUpdate))
	return metricsTime.After(lastUpdate)
}

// updateThingState updates the tracking state for a thing
func (z *zcsazzurroReceiver) updateThingState(thingKey string, metricsTime time.Time) {
	z.logger.Debug("Updating thing state",
		zap.String("thingKey", thingKey),
		zap.Time("metricsTime", metricsTime))
	z.cache.Add(thingKey, metricsTime)
}

func (z *zcsazzurroReceiver) Start(ctx context.Context, host component.Host) error {
	_ctx, cancel := context.WithCancel(ctx)
	z.cancel = cancel

	interval, _ := time.ParseDuration(z.config.Interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-_ctx.Done():
				return
			case <-ticker.C:
				realtimeDataResponse, err := z.scraper.Scrape(z.config.ThingKey)
				if err != nil {
					z.logger.Error("Error scraping zcsazzurro", zap.Error(err))
					continue
				}

				if !realtimeDataResponse.RealtimeData.Success {
					z.logger.Error("Failed to fetch realtime data", zap.Any("response", realtimeDataResponse))
					continue
				}

				for _, v := range realtimeDataResponse.RealtimeData.Params.Value {
					for thingKey, metrics := range v {
						if !z.shouldProcessThing(thingKey, metrics.LastUpdate) {
							z.logger.Debug("Skipping thing - no new data",
								zap.String("thingKey", thingKey),
								zap.Time("lastUpdate", metrics.LastUpdate))
							continue
						}

						processedMetrics, err := z.marshaler.UnmarshalMetrics(thingKey, metrics)
						if err != nil {
							z.logger.Error("Error unmarshalling zcsazzurro metrics", zap.Error(err))
							continue
						}

						if err := z.consumer.ConsumeMetrics(_ctx, processedMetrics); err != nil {
							z.logger.Error("Error consuming zcsazzurro metrics", zap.Error(err))
							continue
						}

						// Only update state after successful processing
						z.updateThingState(thingKey, metrics.LastUpdate)
						z.logger.Debug("Cache keys", zap.Any("keys", z.cache.Keys()))

						z.logger.Info("Successfully processed metrics",
							zap.String("thingKey", thingKey),
							zap.Time("lastUpdate", metrics.LastUpdate))
					}
				}
			}
		}
	}()

	return nil
}

func (z *zcsazzurroReceiver) Shutdown(ctx context.Context) error {
	z.logger.Info("Shutting down zcsazzurro receiver")
	if z.cancel != nil {
		z.cancel()
	}

	return nil
}
