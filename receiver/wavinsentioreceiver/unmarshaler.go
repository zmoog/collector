package wavinsentioreceiver

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	ws2 "github.com/zmoog/ws/v2/ws"
)

const (
	scopeName   = "github.com/zmoog/collector/receiver/wavinsentioreceiver"
	scopeVerion = "v0.2.0"
)

type devicesUnmarshaler struct {
	logger *zap.Logger
}

func (u *devicesUnmarshaler) UnmarshalMetrics(devices []ws2.Device) (pmetric.Metrics, error) {
	u.logger.Info("Unmarshalling devices", zap.Int("device_count", len(devices)))

	timestamp := pcommon.Timestamp(time.Now().UnixNano())
	m := pmetric.NewMetrics()

	for _, device := range devices {
		resourceMetrics := m.ResourceMetrics().AppendEmpty()
		resource := resourceMetrics.Resource()

		resource.Attributes().PutStr("wavinsentio.device.name", device.Name)
		resource.Attributes().PutStr("wavinsentio.device.serial_number", device.SerialNumber)
		resource.Attributes().PutStr("wavinsentio.device.type", device.Type)
		resource.Attributes().PutStr("wavinsentio.device.hc_mode", device.HcMode)
		resource.Attributes().PutStr("wavinsentio.device.firmware.available", device.FirmwareAvailable)
		resource.Attributes().PutStr("wavinsentio.device.firmware.installed", device.FirmwareInstalled)

		scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(scopeName)
		scopeMetrics.Scope().SetVersion(scopeVerion)

		metrics := scopeMetrics.Metrics()

		deviceLastHeartbeat := metrics.AppendEmpty()
		deviceLastHeartbeat.SetName("wavinsentio.device.last_heartbeat")
		deviceLastHeartbeat.SetDescription("Device last heartbeat timestamp")
		deviceLastHeartbeat.SetUnit("ns")
		deviceLastHeartbeatDataPoint := deviceLastHeartbeat.SetEmptyGauge().DataPoints().AppendEmpty()
		deviceLastHeartbeatDataPoint.SetIntValue(device.LastHeartbeat.UnixNano())
		deviceLastHeartbeatDataPoint.SetTimestamp(timestamp)

		if device.LastConfig.Sentio.OutdoorTemperatureSensors != nil {
			for _, sensor := range device.LastConfig.Sentio.OutdoorTemperatureSensors {
				outdoorTemp := metrics.AppendEmpty()
				outdoorTemp.SetName("wavinsentio.device.outdoor_temperature")
				outdoorTemp.SetDescription("Outdoor temperature from sensor")
				outdoorTemp.SetUnit("°C")
				outdoorTempDataPoint := outdoorTemp.SetEmptyGauge().DataPoints().AppendEmpty()
				outdoorTempDataPoint.SetDoubleValue(sensor.OutdoorTemperature)
				outdoorTempDataPoint.SetTimestamp(timestamp)
				outdoorTempDataPoint.Attributes().PutStr("wavinsentio.sensor.id", sensor.ID)
			}
		}

		for _, room := range device.LastConfig.Sentio.Rooms {
			roomResourceMetrics := m.ResourceMetrics().AppendEmpty()
			roomResource := roomResourceMetrics.Resource()

			roomResource.Attributes().PutStr("wavinsentio.device.name", device.Name)
			roomResource.Attributes().PutStr("wavinsentio.device.serial_number", device.SerialNumber)
			roomResource.Attributes().PutStr("wavinsentio.room.id", room.ID)
			roomResource.Attributes().PutStr("wavinsentio.room.title", room.Title)
			roomResource.Attributes().PutStr("wavinsentio.room.temperature_state", room.TemperatureState)
			roomResource.Attributes().PutStr("wavinsentio.room.vacation_mode", room.VacationMode)
			roomResource.Attributes().PutStr("wavinsentio.room.lock_mode", room.LockMode)

			roomScopeMetrics := roomResourceMetrics.ScopeMetrics().AppendEmpty()
			roomScopeMetrics.Scope().SetName(scopeName)
			roomScopeMetrics.Scope().SetVersion(scopeVerion)

			roomMetrics := roomScopeMetrics.Metrics()

			airTemp := roomMetrics.AppendEmpty()
			airTemp.SetName("wavinsentio.room.temperature.air")
			airTemp.SetDescription("Room air temperature")
			airTemp.SetUnit("°C")
			airTempDataPoint := airTemp.SetEmptyGauge().DataPoints().AppendEmpty()
			airTempDataPoint.SetDoubleValue(room.AirTemperature)
			airTempDataPoint.SetTimestamp(timestamp)

			setpointTemp := roomMetrics.AppendEmpty()
			setpointTemp.SetName("wavinsentio.room.temperature.setpoint")
			setpointTemp.SetDescription("Room setpoint temperature")
			setpointTemp.SetUnit("°C")
			setpointTempDataPoint := setpointTemp.SetEmptyGauge().DataPoints().AppendEmpty()
			setpointTempDataPoint.SetDoubleValue(room.SetpointTemperature)
			setpointTempDataPoint.SetTimestamp(timestamp)

			humidity := roomMetrics.AppendEmpty()
			humidity.SetName("wavinsentio.room.humidity")
			humidity.SetDescription("Room humidity")
			humidity.SetUnit("%")
			humidityDataPoint := humidity.SetEmptyGauge().DataPoints().AppendEmpty()
			humidityDataPoint.SetDoubleValue(room.Humidity)
			humidityDataPoint.SetTimestamp(timestamp)

			if room.DehumidifierState != "" {
				dehumidifierState := roomMetrics.AppendEmpty()
				dehumidifierState.SetName("wavinsentio.room.dehumidifier.state")
				dehumidifierState.SetDescription("Dehumidifier state")
				dehumidifierState.SetUnit("")
				dehumidifierStateDataPoint := dehumidifierState.SetEmptyGauge().DataPoints().AppendEmpty()
				stateValue := 0.0
				if room.DehumidifierState == "on" {
					stateValue = 1.0
				}
				dehumidifierStateDataPoint.SetDoubleValue(stateValue)
				dehumidifierStateDataPoint.SetTimestamp(timestamp)
				dehumidifierStateDataPoint.Attributes().PutStr("wavinsentio.dehumidifier.state", room.DehumidifierState)
			}
		}
	}

	return m, nil
}
