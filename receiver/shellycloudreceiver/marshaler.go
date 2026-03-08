package shellycloudreceiver

import (
	"fmt"
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

func (m *shellyMarshaler) MarshalMetrics(devices []deviceData) (pmetric.Metrics, error) {
	md := pmetric.NewMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, d := range devices {
		if d.status == nil {
			continue
		}

		// Gen2: switch:N components
		for channel, sw := range d.status.Switches {
			rm := md.ResourceMetrics().AppendEmpty()
			m.setResourceAttrs(rm.Resource(), d)

			sm := rm.ScopeMetrics().AppendEmpty()
			sm.Scope().SetName(scopeName)
			sm.Scope().SetVersion(scopeVersion)

			m.addSwitchState(sm, channel, sw.Output, now)
			m.addGaugeFloat(sm, "shelly.switch.power", "Active power", "W", channel, sw.APower, now)
			m.addGaugeFloat(sm, "shelly.switch.voltage", "RMS voltage", "V", channel, sw.Voltage, now)
			m.addGaugeFloat(sm, "shelly.switch.current", "RMS current", "A", channel, sw.Current, now)
			m.addGaugeFloat(sm, "shelly.switch.frequency", "AC frequency", "Hz", channel, sw.Freq, now)
			m.addSumFloat(sm, "shelly.switch.energy", "Total energy consumed", "Wh", channel, sw.AEnergy.Total, now)
			if sw.Temperature.TC != 0 {
				m.addGaugeFloat(sm, "shelly.device.temperature", "Device internal temperature", "Cel", channel, sw.Temperature.TC, now)
			}
		}

		// Gen1: meters + relays
		if len(d.status.Meters) > 0 {
			rm := md.ResourceMetrics().AppendEmpty()
			m.setResourceAttrs(rm.Resource(), d)

			sm := rm.ScopeMetrics().AppendEmpty()
			sm.Scope().SetName(scopeName)
			sm.Scope().SetVersion(scopeVersion)

			for i, meter := range d.status.Meters {
				channel := fmt.Sprintf("%d", i)
				m.addGaugeFloat(sm, "shelly.switch.power", "Active power", "W", channel, meter.Power, now)
				m.addSumFloat(sm, "shelly.switch.energy", "Total energy consumed", "Wh", channel, meter.Total, now)
				if i < len(d.status.Relays) {
					m.addSwitchState(sm, channel, d.status.Relays[i].IsOn, now)
				}
			}

			if d.status.Temperature != 0 {
				m.addGaugeFloat(sm, "shelly.device.temperature", "Device internal temperature", "Cel", "0", d.status.Temperature, now)
			}
		}
	}

	return md, nil
}

func (m *shellyMarshaler) setResourceAttrs(res pcommon.Resource, d deviceData) {
	res.Attributes().PutStr("shelly.device.id", d.info.ID)
	res.Attributes().PutStr("shelly.device.name", d.info.Name)
	res.Attributes().PutStr("shelly.device.model", d.info.Type)
	res.Attributes().PutStr("shelly.device.room", d.room)
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
