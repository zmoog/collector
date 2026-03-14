package shellycloudreceiver

import (
	"context"
	"time"

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
	channels, rooms, err := s.client.ListDevices()
	if err != nil {
		return pmetric.NewMetrics(), err
	}
	s.settings.Logger.Info("Fetched Shelly devices", zap.Int("channels", len(channels)))

	// Fetch status once per physical device (multi-channel devices share a base ID).
	// A delay between calls avoids Shelly Cloud rate limiting.
	statusByBaseID := make(map[string]*DeviceStatus)
	first := true
	for _, ch := range channels {
		if !ch.CloudOnline {
			continue
		}
		if _, already := statusByBaseID[ch.BaseID]; already {
			continue
		}
		if !first {
			time.Sleep(s.cfg.RequestDelay)
		}
		first = false

		status, err := s.client.GetDeviceStatus(ch.BaseID)
		if err != nil {
			s.settings.Logger.Error("Failed to get device status",
				zap.String("id", ch.BaseID),
				zap.Error(err))
			statusByBaseID[ch.BaseID] = nil
			continue
		}
		if status == nil {
			s.settings.Logger.Debug("Device offline per status response",
				zap.String("id", ch.BaseID))
		}
		statusByBaseID[ch.BaseID] = status
	}

	// Build one deviceData per channel entry.
	var data []deviceData
	for _, ch := range channels {
		if !ch.CloudOnline {
			s.settings.Logger.Debug("Skipping offline device",
				zap.String("id", ch.ID),
				zap.String("name", ch.Name))
			continue
		}
		status := statusByBaseID[ch.BaseID]
		if status == nil {
			continue
		}
		roomName := ""
		if room, ok := rooms[ch.RoomID]; ok {
			roomName = room.Name
		}
		data = append(data, deviceData{
			info:   ch,
			room:   roomName,
			status: status,
		})
	}

	return s.marshaler.MarshalMetrics(data)
}
