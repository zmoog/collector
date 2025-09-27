package wavinsentioreceiver

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"go.opentelemetry.io/collector/component"
// 	"go.opentelemetry.io/collector/consumer"
// 	"go.uber.org/zap"
// )

// type wavinsentioReceiver struct {
// 	cancel              context.CancelFunc
// 	logger              *zap.Logger
// 	consumer            consumer.Metrics
// 	config              *Config
// 	scraper             *Scraper
// 	locationUnmarshaler *locationUnmarshaler
// 	roomUnmarshaler     *roomUnmarshaler
// }

// func (w *wavinsentioReceiver) Start(ctx context.Context, host component.Host) error {
// 	w.logger.Info("Starting wavinsentio receiver")

// 	_ctx, cancel := context.WithCancel(ctx)
// 	w.cancel = cancel

// 	interval, err := time.ParseDuration(w.config.Interval)
// 	if err != nil {
// 		return fmt.Errorf("invalid interval: %w", err)
// 	}

// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-_ctx.Done():
// 				return
// 			case <-ticker.C:
// 				w.logger.Info("Scraping data from wavinsentio")

// 				results, err := w.scraper.Scrape()
// 				if err != nil {
// 					w.logger.Error("Error scraping wavinsentio", zap.Error(err))
// 					continue
// 				}

// 				for _, r := range results {
// 					locationMetrics, err := w.locationUnmarshaler.UnmarshalMetrics(r.Location)
// 					if err != nil {
// 						w.logger.Error("Error unmarshalling location metrics", zap.Error(err))
// 						continue
// 					}

// 					if err := w.consumer.ConsumeMetrics(_ctx, locationMetrics); err != nil {
// 						w.logger.Error("Error consuming location metrics", zap.Error(err))
// 					}

// 					roomMetrics, err := w.roomUnmarshaler.UnmarshalMetrics(r.Rooms)
// 					if err != nil {
// 						w.logger.Error("Error unmarshalling room metrics", zap.Error(err))
// 						continue
// 					}

// 					if err := w.consumer.ConsumeMetrics(_ctx, roomMetrics); err != nil {
// 						w.logger.Error("Error consuming room metrics", zap.Error(err))
// 					}
// 				}
// 			}
// 		}
// 	}()

// 	return nil
// }

// func (w *wavinsentioReceiver) Shutdown(ctx context.Context) error {
// 	w.logger.Info("Shutting down wavinsentio receiver")
// 	if w.cancel != nil {
// 		w.cancel()
// 	}

// 	return nil
// }
