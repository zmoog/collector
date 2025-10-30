package zcsazzurroreceiver

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/zmoog/zcs/azzurro"
)

const (
	scopeName    = "github.com/zmoog/collector/receiver/zcsazzurroreceiver"
	scopeVersion = "v0.1.0"
)

type azzurroRealtimeDataMarshaler struct {
	logger *zap.Logger
}

func newAzzurroRealtimeDataMarshaler(logger *zap.Logger) *azzurroRealtimeDataMarshaler {
	return &azzurroRealtimeDataMarshaler{logger: logger}
}

func (m *azzurroRealtimeDataMarshaler) addGaugeIntMetric(scopeMetrics pmetric.ScopeMetrics, name, desc, unit string, value int, timestamp pcommon.Timestamp) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetIntValue(int64(value))
	dp.SetTimestamp(timestamp)
}

func (m *azzurroRealtimeDataMarshaler) addGaugeFloatMetric(scopeMetrics pmetric.ScopeMetrics, name, desc, unit string, value float64, timestamp pcommon.Timestamp) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(value)
	dp.SetTimestamp(timestamp)
}

// func (m *azzurroRealtimeDataMarshaler) addSumIntMetric(scopeMetrics pmetric.ScopeMetrics, name, desc, unit string, value int, timestamp pcommon.Timestamp) {
// 	metric := scopeMetrics.Metrics().AppendEmpty()
// 	metric.SetName(name)
// 	metric.SetDescription(desc)
// 	metric.SetUnit(unit)
// 	sum := metric.SetEmptySum()
// 	sum.SetIsMonotonic(true)
// 	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
// 	dp := sum.DataPoints().AppendEmpty()
// 	dp.SetIntValue(int64(value))
// 	dp.SetTimestamp(timestamp)
// }

func (m *azzurroRealtimeDataMarshaler) addSumFloatMetric(scopeMetrics pmetric.ScopeMetrics, name, desc, unit string, value float64, timestamp pcommon.Timestamp, startTimestamp pcommon.Timestamp) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp := sum.DataPoints().AppendEmpty()
	dp.SetDoubleValue(value)
	dp.SetTimestamp(timestamp)
	dp.SetStartTimestamp(startTimestamp)
}

func (m *azzurroRealtimeDataMarshaler) addSumIntMetric(scopeMetrics pmetric.ScopeMetrics, name, desc, unit string, value int, timestamp pcommon.Timestamp, startTimestamp pcommon.Timestamp) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(desc)
	metric.SetUnit(unit)
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp := sum.DataPoints().AppendEmpty()
	dp.SetIntValue(int64(value))
	dp.SetTimestamp(timestamp)
	dp.SetStartTimestamp(startTimestamp)
}

func (m *azzurroRealtimeDataMarshaler) UnmarshalMetrics(thingKey string, metrics azzurro.InverterMetrics) (pmetric.Metrics, error) {
	md := pmetric.NewMetrics()

	resourceMetrics := md.ResourceMetrics().AppendEmpty()

	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName(scopeName)
	scopeMetrics.Scope().SetVersion(scopeVersion)

	// ----------------------------------------------------------------
	// Resource attributes
	// ----------------------------------------------------------------
	resource := resourceMetrics.Resource()
	resource.Attributes().PutStr("thing_key", thingKey)

	// ----------------------------------------------------------------
	// Timestamp
	// ----------------------------------------------------------------
	timestamp := pcommon.Timestamp(metrics.LastUpdate.UnixNano())

	// I assume the start of today is the start of the daily metrics
	startOfTodayTimestamp := pcommon.Timestamp(metrics.LastUpdate.Truncate(24 * time.Hour).UnixNano())

	// Parse thingFind for total metrics start timestamp
	thingFindTime, err := time.Parse("2006-01-02T15:04:05Z", metrics.ThingFind)
	if err != nil {
		m.logger.Warn("Failed to parse thingFind timestamp, using daily start", zap.String("thingFind", metrics.ThingFind), zap.Error(err))
		thingFindTime = metrics.LastUpdate.Truncate(24 * time.Hour)
	}

	// I assume thingFind is the discovery timestamp, when the thing was
	// first installed. This is the start of the total cumulative metrics.
	thingDiscoveryTimestamp := pcommon.Timestamp(thingFindTime.UnixNano())

	// Power metrics
	// ----------------------------------------------------------------
	m.addGaugeFloatMetric(scopeMetrics, "power_autoconsuming", "Power autoconsuming", "W", metrics.PowerAutoconsuming, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_charging", "Power charging", "W", metrics.PowerCharging, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_consuming", "Power consuming", "W", metrics.PowerConsuming, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_discharging", "Power discharging", "W", metrics.PowerDischarging, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_exporting", "Power exporting", "W", metrics.PowerExporting, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_generating", "Power generating", "W", metrics.PowerGenerating, timestamp)
	m.addGaugeFloatMetric(scopeMetrics, "power_importing", "Power importing", "W", metrics.PowerImporting, timestamp)

	// ----------------------------------------------------------------
	// Battery metrics
	// ----------------------------------------------------------------
	m.addGaugeIntMetric(scopeMetrics, "battery_soc", "Battery SOC", "%", metrics.BatterySoC, timestamp)
	m.addSumIntMetric(scopeMetrics, "battery_cycletime_total", "Total battery cycletime", "cycles", metrics.BatteryCycletime, timestamp, thingDiscoveryTimestamp)

	// ----------------------------------------------------------------
	// Energy metrics
	// ----------------------------------------------------------------

	// Gauge metrics for current energy values
	m.addSumFloatMetric(scopeMetrics, "energy_autoconsuming", "Energy autoconsuming", "kWh", metrics.EnergyAutoconsuming, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_charging", "Energy charging", "kWh", metrics.EnergyCharging, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_consuming", "Energy consuming", "kWh", metrics.EnergyConsuming, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_discharging", "Energy discharging", "kWh", metrics.EnergyDischarging, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_exporting", "Energy exporting", "kWh", metrics.EnergyExporting, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_generating", "Energy generating", "kWh", metrics.EnergyGenerating, timestamp, startOfTodayTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_importing", "Energy importing", "kWh", metrics.EnergyImporting, timestamp, startOfTodayTimestamp)

	// Sum metrics for total energy values (cumulative) - lifetime totals since thingFind
	m.addSumFloatMetric(scopeMetrics, "energy_autoconsuming_total", "Energy autoconsuming total", "kWh", metrics.EnergyAutoconsumingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_charging_total", "Energy charging total", "kWh", metrics.EnergyChargingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_consuming_total", "Energy consuming total", "kWh", metrics.EnergyConsumingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_discharging_total", "Energy discharging total", "kWh", metrics.EnergyDischargingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_exporting_total", "Energy exporting total", "kWh", metrics.EnergyExportingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_generating_total", "Energy generating total", "kWh", metrics.EnergyGeneratingTotal, timestamp, thingDiscoveryTimestamp)
	m.addSumFloatMetric(scopeMetrics, "energy_importing_total", "Energy importing total", "kWh", metrics.EnergyImportingTotal, timestamp, thingDiscoveryTimestamp)

	return md, nil
}
