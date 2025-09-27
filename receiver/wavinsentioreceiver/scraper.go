package wavinsentioreceiver

import (
	"context"

	"github.com/zmoog/ws/v2/ws"
	"github.com/zmoog/ws/v2/ws/identity"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

type wavinsentioScraper struct {
	cfg                *Config
	settings           component.TelemetrySettings
	client             *ws.Client
	devicesUnmarshaler *devicesUnmarshaler
}

func newScraper(cfg *Config, settings receiver.Settings) *wavinsentioScraper {
	return &wavinsentioScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		devicesUnmarshaler: &devicesUnmarshaler{
			logger: settings.Logger,
		},
	}
}

func (s *wavinsentioScraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	devices, err := s.client.ListDevices()
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	return s.devicesUnmarshaler.UnmarshalMetrics(devices)
}

func (s *wavinsentioScraper) start(_ context.Context, host component.Host) (err error) {
	identityManager := identity.NewManager(
		s.cfg.Username,
		s.cfg.Password,
		s.cfg.WebApiKey,
	)
	s.client = ws.NewClient(identityManager, s.cfg.Endpoint)
	return nil
}
