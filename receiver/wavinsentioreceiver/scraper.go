package wavinsentioreceiver

import (
	"context"

	"github.com/zmoog/ws/v2/ws"
	"github.com/zmoog/ws/v2/ws/identity"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

// wavinsentioScraper is the struct that contains the Wavin Sentio scraper.
type wavinsentioScraper struct {
	cfg                *Config
	settings           component.TelemetrySettings
	client             *ws.Client
	devicesUnmarshaler *devicesUnmarshaler
}

// newScraper is the function that creates a new Wavin Sentio scraper.
func newScraper(cfg *Config, settings receiver.Settings) *wavinsentioScraper {
	return &wavinsentioScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		devicesUnmarshaler: &devicesUnmarshaler{
			logger: settings.Logger,
		},
	}
}

// scrape is the main function that scrapes the data from the Wavin Sentio API.
func (s *wavinsentioScraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	devices, err := s.client.ListDevices()
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	return s.devicesUnmarshaler.UnmarshalMetrics(devices)
}

// start is the function that starts the Wavin Sentio scraper.
func (s *wavinsentioScraper) start(_ context.Context, host component.Host) (err error) {
	identityManager := identity.NewInMemoryManager(
		identity.Config{
			Username:  s.cfg.Username,
			Password:  s.cfg.Password,
			WebApiKey: s.cfg.WebApiKey,
		},
	)
	s.client = ws.NewClient(identityManager, s.cfg.Endpoint)
	return nil
}
