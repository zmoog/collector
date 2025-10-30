package zcsazzurroreceiver

import (
	"context"
	"time"

	"github.com/elastic/go-freelru"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/zmoog/zcs/azzurro"
)

// zcsazzurroScraper is the struct that contains the ZCS Azzurro scraper.
type zcsazzurroScraper struct {
	cfg       *Config
	settings  component.TelemetrySettings
	client    *azzurro.Client
	marshaler *azzurroRealtimeDataMarshaler
	cache     *freelru.SyncedLRU[string, time.Time]
}

// newScraper is the function that creates a new ZCS Azzurro scraper.
func newScraper(cfg *Config, settings receiver.Settings, cache *freelru.SyncedLRU[string, time.Time]) *zcsazzurroScraper {
	client := azzurro.NewClient(cfg.AuthKey, cfg.ClientID)
	return &zcsazzurroScraper{
		cfg:       cfg,
		settings:  settings.TelemetrySettings,
		client:    client,
		marshaler: newAzzurroRealtimeDataMarshaler(settings.Logger),
		cache:     cache,
	}
}

// start is the function that starts the ZCS Azzurro scraper.
func (s *zcsazzurroScraper) start(_ context.Context, host component.Host) error {
	// Nothing special needed for start - client is already initialized
	return nil
}

// scrape is the main function that scrapes the data from the ZCS Azzurro API.
func (s *zcsazzurroScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	realtimeDataResponse, err := s.client.FetchRealtimeData(s.cfg.ThingKey)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	if !realtimeDataResponse.RealtimeData.Success {
		s.settings.Logger.Error("Failed to fetch realtime data", zap.Any("response", realtimeDataResponse))
		return pmetric.NewMetrics(), nil // Return empty metrics instead of error for non-critical failures
	}

	// Aggregate all metrics from all things into a single pmetric.Metrics
	allMetrics := pmetric.NewMetrics()
	
	for _, v := range realtimeDataResponse.RealtimeData.Params.Value {
		for thingKey, metrics := range v {
			if !s.shouldProcessThing(thingKey, metrics.LastUpdate) {
				s.settings.Logger.Debug("Skipping thing - no new data",
					zap.String("thingKey", thingKey),
					zap.Time("lastUpdate", metrics.LastUpdate))
				continue
			}

			processedMetrics, err := s.marshaler.UnmarshalMetrics(thingKey, metrics)
			if err != nil {
				s.settings.Logger.Error("Error unmarshalling zcsazzurro metrics", zap.Error(err))
				continue
			}

			// Only update state after successful processing
			s.updateThingState(thingKey, metrics.LastUpdate)
			s.settings.Logger.Debug("Cache keys", zap.Any("keys", s.cache.Keys()))

			s.settings.Logger.Info("Successfully processed metrics",
				zap.String("thingKey", thingKey),
				zap.Time("lastUpdate", metrics.LastUpdate))

			// Merge metrics into the aggregated result
			processedMetrics.ResourceMetrics().MoveAndAppendTo(allMetrics.ResourceMetrics())
		}
	}

	return allMetrics, nil
}

// shouldProcessThing checks if we should process metrics for this thing
// Returns true if:
// - We haven't seen this thing before, OR
// - The metrics timestamp is newer than what we last processed
func (s *zcsazzurroScraper) shouldProcessThing(thingKey string, metricsTime time.Time) bool {
	s.settings.Logger.Debug("Checking if should process thing",
		zap.String("thingKey", thingKey),
		zap.Time("metricsTime", metricsTime))

	s.settings.Logger.Debug("Cache keys", zap.Any("keys", s.cache.Keys()))
	lastUpdate, exists := s.cache.Get(thingKey)
	if !exists {
		s.settings.Logger.Debug("Thing not seen before, processing", zap.String("thingKey", thingKey))
		return true
	}

	s.settings.Logger.Debug("Thing seen before, checking last update",
		zap.String("thingKey", thingKey),
		zap.Time("lastUpdate", lastUpdate))
	return metricsTime.After(lastUpdate)
}

// updateThingState updates the tracking state for a thing
func (s *zcsazzurroScraper) updateThingState(thingKey string, metricsTime time.Time) {
	s.settings.Logger.Debug("Updating thing state",
		zap.String("thingKey", thingKey),
		zap.Time("metricsTime", metricsTime))
	s.cache.Add(thingKey, metricsTime)
}

// Legacy scraper struct and functions for backward compatibility
// These are preserved but not used in the new implementation

func NewScraper(clientID, authKey, thingKey string, logger *zap.Logger) *Scraper {
	client := azzurro.NewClient(authKey, clientID)
	return &Scraper{
		client: client,
		logger: logger,
	}
}

type Scraper struct {
	logger *zap.Logger
	client *azzurro.Client
}

func (s *Scraper) Scrape(thingKey string) (azzurro.RealtimeDataResponse, error) {
	response, err := s.client.FetchRealtimeData(thingKey)
	if err != nil {
		return azzurro.RealtimeDataResponse{}, err
	}

	return response, nil
}
