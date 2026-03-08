package shellycloudreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type shellyScraper struct {
	cfg       *Config
	settings  component.TelemetrySettings
	client    *Client
	marshaler *shellyMarshaler
}

func newScraper(cfg *Config, settings receiver.Settings) *shellyScraper {
	return &shellyScraper{
		cfg:       cfg,
		settings:  settings.TelemetrySettings,
		client:    newClient(cfg.ServerURL, cfg.AuthKey),
		marshaler: newMarshaler(settings.Logger),
	}
}

func (s *shellyScraper) start(_ context.Context, _ component.Host) error {
	return nil
}

func (s *shellyScraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	devices, rooms, err := s.client.ListDevices()
	if err != nil {
		return pmetric.NewMetrics(), err
	}
	s.settings.Logger.Info("Fetched Shelly devices", zap.Int("count", len(devices)))

	var data []deviceData
	for _, device := range devices {
		if !device.Online {
			s.settings.Logger.Debug("Skipping offline device",
				zap.String("id", device.ID),
				zap.String("name", device.Name))
			continue
		}

		status, err := s.client.GetDeviceStatus(device.ID)
		if err != nil {
			s.settings.Logger.Error("Failed to get device status",
				zap.String("id", device.ID),
				zap.String("name", device.Name),
				zap.Error(err))
			continue
		}

		roomName := ""
		if room, ok := rooms[device.RoomID]; ok {
			roomName = room.Name
		}

		data = append(data, deviceData{
			info:   device,
			room:   roomName,
			status: status,
		})
	}

	return s.marshaler.MarshalMetrics(data)
}
