package shellycloudreceiver

import (
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const (
	scopeName    = "github.com/zmoog/collector/receiver/shellycloudreceiver"
	scopeVersion = "v0.1.0"
)

type deviceData struct {
	info   DeviceInfo
	room   string
	status *DeviceStatus
}

type shellyMarshaler struct {
	logger *zap.Logger
}

func newMarshaler(logger *zap.Logger) *shellyMarshaler {
	return &shellyMarshaler{logger: logger}
}

// MarshalMetrics emits one ResourceMetrics per channel entry.
// Each channel already has its own name and room from the device list,
// so the channel index is used only to select the right status data.
func (m *shellyMarshaler) MarshalMetrics(devices []deviceData) (pmetric.Metrics, error) {
	md := pmetric.NewMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, d := range devices {
		if d.status == nil {
			continue
		}

		channel := strconv.Itoa(d.info.Channel)

		rm := md.ResourceMetrics().AppendEmpty()
		m.setResourceAttrs(rm.Resource(), d)

		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(scopeName)
		sm.Scope().SetVersion(scopeVersion)

		if d.info.Gen == 1 {
			m.marshalGen1(sm, d, channel, now)
		} else {
			m.marshalGen2(sm, d, channel, now)
		}
	}

	return md, nil
}

func (m *shellyMarshaler) marshalGen1(sm pmetric.ScopeMetrics, d deviceData, channel string, now pcommon.Timestamp) {
	ch := d.info.Channel
	if ch >= len(d.status.Meters) {
		m.logger.Debug("No Gen1 meter for channel",
			zap.String("id", d.info.ID), zap.Int("channel", ch))
		return
	}
	meter := d.status.Meters[ch]
	m.addGaugeFloat(sm, "shelly.switch.power", "Active power", "W", channel, meter.Power, now)
	m.addSumFloat(sm, "shelly.switch.energy", "Total energy consumed", "Wh", channel, meter.Total, now)

	if ch < len(d.status.Relays) {
		m.addSwitchState(sm, channel, d.status.Relays[ch].IsOn, now)
	}
	if d.status.Temperature != 0 {
		m.addGaugeFloat(sm, "shelly.device.temperature", "Device internal temperature", "Cel", channel, d.status.Temperature, now)
	}
	if d.status.Wifi.RSSI != 0 {
		m.addGaugeInt(sm, "shelly.wifi.rssi", "WiFi signal strength", "dBm", channel, d.status.Wifi.RSSI, now)
	}
}

func (m *shellyMarshaler) marshalGen2(sm pmetric.ScopeMetrics, d deviceData, channel string, now pcommon.Timestamp) {
	sw, ok := d.status.Switches[channel]
	if !ok {
		m.logger.Debug("No Gen2+ switch status for channel",
			zap.String("id", d.info.ID), zap.String("channel", channel))
		return
	}
	m.addSwitchState(sm, channel, sw.Output, now)
	m.addGaugeFloat(sm, "shelly.switch.power", "Active power", "W", channel, sw.APower, now)
	m.addGaugeFloat(sm, "shelly.switch.voltage", "RMS voltage", "V", channel, sw.Voltage, now)
	m.addGaugeFloat(sm, "shelly.switch.current", "RMS current", "A", channel, sw.Current, now)
	m.addGaugeFloat(sm, "shelly.switch.frequency", "AC frequency", "Hz", channel, sw.Freq, now)
	m.addSumFloat(sm, "shelly.switch.energy", "Total energy consumed", "Wh", channel, sw.AEnergy.Total, now)
	if sw.Temperature.TC != 0 {
		m.addGaugeFloat(sm, "shelly.device.temperature", "Device internal temperature", "Cel", channel, sw.Temperature.TC, now)
	}
	if d.status.Wifi.RSSI != 0 {
		m.addGaugeInt(sm, "shelly.wifi.rssi", "WiFi signal strength", "dBm", channel, d.status.Wifi.RSSI, now)
	}
}

func (m *shellyMarshaler) setResourceAttrs(res pcommon.Resource, d deviceData) {
	res.Attributes().PutStr("shelly.device.id", d.info.ID)
	res.Attributes().PutStr("shelly.device.name", d.info.Name)
	res.Attributes().PutStr("shelly.device.model", d.info.Type)
	res.Attributes().PutStr("shelly.device.room", d.room)
	if d.status.Wifi.SSID != "" {
		res.Attributes().PutStr("shelly.wifi.ssid", d.status.Wifi.SSID)
	}
	if d.status.Wifi.IP != "" {
		res.Attributes().PutStr("shelly.wifi.ip", d.status.Wifi.IP)
	}
}

func (m *shellyMarshaler) addSwitchState(sm pmetric.ScopeMetrics, channel string, on bool, ts pcommon.Timestamp) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("shelly.switch.state")
	metric.SetDescription("Switch output state (1=on, 0=off)")
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.Attributes().PutStr("shelly.channel", channel)
	dp.SetTimestamp(ts)
	if on {
		dp.SetIntValue(1)
	} else {
		dp.SetIntValue(0)
	}
}

func (m *shellyMarshaler) addGaugeInt(sm pmetric.ScopeMetrics, name, desc, unit, channel string, value int, ts pcommon.Timestamp) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.Attributes().PutStr("shelly.channel", channel)
	dp.SetIntValue(int64(value))
	dp.SetTimestamp(ts)
}

func (m *shellyMarshaler) addGaugeFloat(sm pmetric.ScopeMetrics, name, desc, unit, channel string, value float64, ts pcommon.Timestamp) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.Attributes().PutStr("shelly.channel", channel)
	dp.SetDoubleValue(value)
	dp.SetTimestamp(ts)
}

func (m *shellyMarshaler) addSumFloat(sm pmetric.ScopeMetrics, name, desc, unit, channel string, value float64, ts pcommon.Timestamp) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp := sum.DataPoints().AppendEmpty()
	dp.Attributes().PutStr("shelly.channel", channel)
	dp.SetDoubleValue(value)
	dp.SetTimestamp(ts)
}
