package toggltrackreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type togglTrackReceiver struct {
	cancel    context.CancelFunc
	logger    *zap.Logger
	consumer  consumer.Logs
	config    *Config
	scraper   *Scraper
	marshaler *timeEntryMarshaler
}

func (t *togglTrackReceiver) Start(ctx context.Context, host component.Host) error {
	t.logger.Info("Starting toggltrack receiver")

	_ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	interval, _ := time.ParseDuration(t.config.Interval)
	lookback, _ := time.ParseDuration(t.config.Lookback)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-_ctx.Done():
				return
			case <-ticker.C:
				// Do something
				t.logger.Info("Doing something")

				entries, err := t.scraper.Scrape(time.Now(), lookback)
				if err != nil {
					t.logger.Error("Error scraping toggltrack", zap.Error(err))
					continue
				}

				t.logger.Info("Scraped toggltrack entries", zap.Int("count", len(entries)))

				if len(entries) == 0 {
					t.logger.Info("No new entries to process")
					continue
				}

				logs, err := t.marshaler.UnmarshalLogs(entries)
				if err != nil {
					t.logger.Error("Error marshaling toggltrack entries", zap.Error(err))
					continue
				}

				if err := t.consumer.ConsumeLogs(_ctx, logs); err != nil {
					t.logger.Error("Error consuming toggltrack logs", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (t *togglTrackReceiver) Shutdown(ctx context.Context) error {
	t.logger.Info("Shutting down toggltrack receiver")
	if t.cancel != nil {
		t.cancel()
	}

	return nil
}
