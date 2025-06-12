package zcsazzurroreceiver

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type thingState struct {
	lastUpdate time.Time
	lastSeen   time.Time
}

type zcsazzurroReceiver struct {
	cancel       context.CancelFunc
	logger       *zap.Logger
	consumer     consumer.Metrics
	config       *Config
	scraper      *Scraper
	marshaler    *azzurroRealtimeDataMarshaler
	thingStates  map[string]*thingState
	thingsMutex  sync.RWMutex
}

// shouldProcessThing checks if we should process metrics for this thing
// Returns true if:
// - We haven't seen this thing before, OR
// - The metrics timestamp is newer than what we last processed
func (z *zcsazzurroReceiver) shouldProcessThing(thingKey string, metricsTime time.Time) bool {
	z.thingsMutex.RLock()
	state, exists := z.thingStates[thingKey]
	z.thingsMutex.RUnlock()
	
	if !exists {
		return true
	}
	
	return metricsTime.After(state.lastUpdate)
}

// updateThingState updates the tracking state for a thing
func (z *zcsazzurroReceiver) updateThingState(thingKey string, metricsTime time.Time) {
	now := time.Now()
	z.thingsMutex.Lock()
	defer z.thingsMutex.Unlock()
	
	if z.thingStates == nil {
		z.thingStates = make(map[string]*thingState)
	}
	
	if state, exists := z.thingStates[thingKey]; exists {
		state.lastUpdate = metricsTime
		state.lastSeen = now
	} else {
		z.thingStates[thingKey] = &thingState{
			lastUpdate: metricsTime,
			lastSeen:   now,
		}
	}
}

// cleanupStaleThings removes things that haven't been seen for a while to prevent memory leaks
func (z *zcsazzurroReceiver) cleanupStaleThings(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	z.thingsMutex.Lock()
	defer z.thingsMutex.Unlock()
	
	for thingKey, state := range z.thingStates {
		if state.lastSeen.Before(cutoff) {
			z.logger.Debug("Removing stale thing from tracking", 
				zap.String("thingKey", thingKey),
				zap.Time("lastSeen", state.lastSeen))
			delete(z.thingStates, thingKey)
		}
	}
}

func (z *zcsazzurroReceiver) Start(ctx context.Context, host component.Host) error {
	_ctx, cancel := context.WithCancel(ctx)
	z.cancel = cancel

	interval, _ := time.ParseDuration(z.config.Interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		// Cleanup ticker to prevent memory leaks - run every hour
		cleanupTicker := time.NewTicker(1 * time.Hour)
		defer cleanupTicker.Stop()

		for {
			select {
			case <-_ctx.Done():
				return
			case <-cleanupTicker.C:
				// Remove things that haven't been seen for 24 hours
				z.cleanupStaleThings(24 * time.Hour)
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
						
						z.logger.Debug("Successfully processed metrics", 
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
