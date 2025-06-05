package wavinsentioreceiver

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/zmoog/ws/ws"
)

const (
	scopeName   = "github.com/zmoog/collector/receiver/wavinsentioreceiver"
	scopeVerion = "v0.1.0"
)

type locationUnmarshaler struct {
	logger *zap.Logger
}

func (u *locationUnmarshaler) UnmarshalMetrics(response ws.Location) (pmetric.Metrics, error) {
	u.logger.Info("Unmarshalling wavinsentio location", zap.Any("response", response))

	md := pmetric.NewMetrics()

	resourceMetrics := md.ResourceMetrics().AppendEmpty()
	resource := resourceMetrics.Resource()

	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName(scopeName)
	scopeMetrics.Scope().SetVersion(scopeVerion)

	// ----------------------------------------------------------------
	// Resource attributes
	// ----------------------------------------------------------------
	resource.Attributes().PutStr("wavinsentio.location.id", response.Ulc)
	resource.Attributes().PutStr("wavinsentio.location.serial_number", fmt.Sprintf("%d", response.SerialNumber))

	// ----------------------------------------------------------------
	// Timestamp
	// ----------------------------------------------------------------
	timestamp := pcommon.Timestamp(time.Now().UnixNano())

	// ----------------------------------------------------------------
	// Metrics
	// ----------------------------------------------------------------
	outdoorTemperature := scopeMetrics.Metrics().AppendEmpty()
	outdoorTemperature.SetName("wavinsentio.location.outdoor_temperature")
	outdoorTemperature.SetDescription("Outdoor temperature")
	outdoorTemperature.SetUnit("°C")
	outdoorTemperatureDataPoint := outdoorTemperature.SetEmptyGauge().DataPoints().AppendEmpty()
	outdoorTemperatureDataPoint.SetDoubleValue(response.Attributes.Outdoor.Temperature)
	outdoorTemperatureDataPoint.SetTimestamp(timestamp)

	return md, nil
}

type roomUnmarshaler struct {
	logger *zap.Logger
}

func (u *roomUnmarshaler) UnmarshalMetrics(rooms []ws.Room) (pmetric.Metrics, error) {
	u.logger.Info("Unmarshalling rooms", zap.Any("rooms", rooms))

	// ----------------------------------------------------------------
	// Timestamp
	// ----------------------------------------------------------------
	timestamp := pcommon.Timestamp(time.Now().UnixNano())

	m := pmetric.NewMetrics()

	for _, room := range rooms {

		resourceMetrics := m.ResourceMetrics().AppendEmpty()

		// ----------------------------------------------------------------
		// Resource attributes
		// ----------------------------------------------------------------

		resource := resourceMetrics.Resource()
		resource.Attributes().PutStr("wavinsentio.room.code", room.Code)
		resource.Attributes().PutStr("wavinsentio.room.name", room.Name)
		resource.Attributes().PutStr("wavinsentio.room.status", room.Status)

		// ----------------------------------------------------------------
		// Scope metrics
		// ----------------------------------------------------------------

		scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(scopeName)
		scopeMetrics.Scope().SetVersion(scopeVerion)

		// ----------------------------------------------------------------
		// Metrics
		// ----------------------------------------------------------------

		metrics := scopeMetrics.Metrics()

		temperatureDesired := metrics.AppendEmpty()
		temperatureDesired.SetName("wavinsentio.room.temperature.desired")
		temperatureDesired.SetDescription("Temperature desired")
		temperatureDesired.SetUnit("°C")
		temperatureDesiredDataPoint := temperatureDesired.SetEmptyGauge().DataPoints().AppendEmpty()
		temperatureDesiredDataPoint.SetDoubleValue(room.TempDesired)
		temperatureDesiredDataPoint.SetTimestamp(timestamp)
		temperatureDesiredDataPoint.Attributes().PutStr("wavinsentio.room.code", room.Code)

		temperatureCurrent := metrics.AppendEmpty()
		temperatureCurrent.SetName("wavinsentio.room.temperature.current")
		temperatureCurrent.SetDescription("Temperature current")
		temperatureCurrent.SetUnit("°C")
		temperatureCurrentDataPoint := temperatureCurrent.SetEmptyGauge().DataPoints().AppendEmpty()
		temperatureCurrentDataPoint.SetDoubleValue(room.TempCurrent)
		temperatureCurrentDataPoint.SetTimestamp(timestamp)
		temperatureCurrentDataPoint.Attributes().PutStr("wavinsentio.room.code", room.Code)

		humidityCurrent := metrics.AppendEmpty()
		humidityCurrent.SetName("wavinsentio.room.humidity.current")
		humidityCurrent.SetDescription("Humidity current")
		humidityCurrent.SetUnit("%")
		humidityCurrentDataPoint := humidityCurrent.SetEmptyGauge().DataPoints().AppendEmpty()
		humidityCurrentDataPoint.SetDoubleValue(room.HumidityCurrent)
		humidityCurrentDataPoint.SetTimestamp(timestamp)
		humidityCurrentDataPoint.Attributes().PutStr("wavinsentio.room.code", room.Code)
	}

	return m, nil
}
