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

// func newScraper(username, password string, logger *zap.Logger) *Scraper {
// 	identityManager := identity.NewManager(username, password)
// 	client := ws.NewClient(identityManager, "https://wavin-api.jablotron.cloud")

// 	return &Scraper{
// 		logger: logger,
// 		client: client,
// 	}
// }

// type Scraper struct {
// 	logger *zap.Logger
// 	client *ws.Client
// }

// // Scrape scrapes the data from the wavinsentio API.
// func (s *Scraper) Scrape() ([]ScrapeResult, error) {
// 	results := []ScrapeResult{}

// 	locations, err := s.client.ListLocations()
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, location := range locations {
// 		rooms, err := s.client.ListRooms(location.Ulc)
// 		if err != nil {
// 			return nil, err
// 		}

// 		results = append(results, ScrapeResult{
// 			Location: location,
// 			Rooms:    rooms,
// 		})
// 	}

// 	return results, nil
// }

// // ScrapeResult is the result of a scrape.
// // It contains a location and all rooms in that location.
// type ScrapeResult struct {
// 	Location ws.Location
// 	Rooms    []ws.Room
// }
