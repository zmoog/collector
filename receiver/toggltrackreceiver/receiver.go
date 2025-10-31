package toggltrackreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// togglTrackScraper is the struct that contains the TogglTrack scraper.
type togglTrackScraper struct {
	cfg       *Config
	settings  component.TelemetrySettings
	scraper   *Scraper
	marshaler *timeEntryMarshaler
}

// newScraper creates a new TogglTrack scraper.
func newScraper(cfg *Config, settings receiver.Settings) *togglTrackScraper {
	return &togglTrackScraper{
		cfg:       cfg,
		settings:  settings.TelemetrySettings,
		scraper:   NewScraper(cfg.APIToken, settings.Logger),
		marshaler: &timeEntryMarshaler{},
	}
}

// start initializes the TogglTrack scraper.
func (s *togglTrackScraper) start(_ context.Context, host component.Host) error {
	s.settings.Logger.Info("Starting toggltrack scraper")
	return nil
}

// scrape is the main function that scrapes the data from the TogglTrack API.
func (s *togglTrackScraper) scrape(ctx context.Context) (plog.Logs, error) {
	lookback, err := time.ParseDuration(s.cfg.Lookback)
	if err != nil {
		s.settings.Logger.Error("Error parsing lookback duration", zap.Error(err))
		return plog.NewLogs(), err
	}

	entries, err := s.scraper.Scrape(time.Now(), lookback)
	if err != nil {
		s.settings.Logger.Error("Error scraping toggltrack", zap.Error(err))
		return plog.NewLogs(), err
	}

	s.settings.Logger.Info("Scraped toggltrack entries", zap.Int("count", len(entries)))

	if len(entries) == 0 {
		s.settings.Logger.Debug("No new entries to process")
		return plog.NewLogs(), nil
	}

	logs, err := s.marshaler.UnmarshalLogs(entries)
	if err != nil {
		s.settings.Logger.Error("Error marshaling toggltrack entries", zap.Error(err))
		return plog.NewLogs(), err
	}

	return logs, nil
}
