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

func (m *azzurroRealtimeDataMarshaler) UnmarshalMetrics(response azzurro.RealtimeDataResponse) (pmetric.Metrics, error) {
	if !response.RealtimeData.Success {
		m.logger.Error("Failed to fetch realtime data", zap.Any("response", response))
		return pmetric.NewMetrics(), nil
	}

	m.logger.Info("Unmarshalling azzurro realtime data response", zap.Any("response", response))
	md := pmetric.NewMetrics()

	for _, v := range response.RealtimeData.Params.Value {
		for thingKey, value := range v {
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
			timestamp := pcommon.Timestamp(value.LastUpdate.UnixNano())
			
			// I assume the start of today is the start of the daily metrics
			startOfTodayTimestamp := pcommon.Timestamp(value.LastUpdate.Truncate(24 * time.Hour).UnixNano())
			
			// Parse thingFind for total metrics start timestamp
			thingFindTime, err := time.Parse("2006-01-02T15:04:05Z", value.ThingFind)
			if err != nil {
				m.logger.Warn("Failed to parse thingFind timestamp, using daily start", zap.String("thingFind", value.ThingFind), zap.Error(err))
				thingFindTime = value.LastUpdate.Truncate(24 * time.Hour)
			}
			// I assume thingFind is the discovery timestamp, when the thing was
			// first installed. This is the start of the total cumulative metrics.
			thingDiscoveryTimestamp := pcommon.Timestamp(thingFindTime.UnixNano())

			// Power metrics
			// ----------------------------------------------------------------
			m.addGaugeFloatMetric(scopeMetrics, "power_autoconsuming", "Power autoconsuming", "W", value.PowerAutoconsuming, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_charging", "Power charging", "W", value.PowerCharging, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_consuming", "Power consuming", "W", value.PowerConsuming, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_discharging", "Power discharging", "W", value.PowerDischarging, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_exporting", "Power exporting", "W", value.PowerExporting, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_generating", "Power generating", "W", value.PowerGenerating, timestamp)
			m.addGaugeFloatMetric(scopeMetrics, "power_importing", "Power importing", "W", value.PowerImporting, timestamp)

			// ----------------------------------------------------------------
			// Battery metrics
			// ----------------------------------------------------------------
			m.addGaugeIntMetric(scopeMetrics, "battery_soc", "Battery SOC", "%", value.BatterySoC, timestamp)
			m.addSumIntMetric(scopeMetrics, "battery_cycletime_total", "Total battery cycletime", "cycles", value.BatteryCycletime, timestamp, thingDiscoveryTimestamp)

			// ----------------------------------------------------------------
			// Energy metrics
			// ----------------------------------------------------------------
			// Gauge metrics for current energy values
			m.addSumFloatMetric(scopeMetrics, "energy_autoconsuming", "Energy autoconsuming", "kWh", value.EnergyAutoconsuming, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_charging", "Energy charging", "kWh", value.EnergyCharging, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_consuming", "Energy consuming", "kWh", value.EnergyConsuming, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_discharging", "Energy discharging", "kWh", value.EnergyDischarging, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_exporting", "Energy exporting", "kWh", value.EnergyExporting, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_generating", "Energy generating", "kWh", value.EnergyGenerating, timestamp, startOfTodayTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_importing", "Energy importing", "kWh", value.EnergyImporting, timestamp, startOfTodayTimestamp)

			// Sum metrics for total energy values (cumulative) - lifetime totals since thingFind
			m.addSumFloatMetric(scopeMetrics, "energy_autoconsuming_total", "Energy autoconsuming total", "kWh", value.EnergyAutoconsumingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_charging_total", "Energy charging total", "kWh", value.EnergyChargingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_consuming_total", "Energy consuming total", "kWh", value.EnergyConsumingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_discharging_total", "Energy discharging total", "kWh", value.EnergyDischargingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_exporting_total", "Energy exporting total", "kWh", value.EnergyExportingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_generating_total", "Energy generating total", "kWh", value.EnergyGeneratingTotal, timestamp, thingDiscoveryTimestamp)
			m.addSumFloatMetric(scopeMetrics, "energy_importing_total", "Energy importing total", "kWh", value.EnergyImportingTotal, timestamp, thingDiscoveryTimestamp)
		}
	}
	return md, nil
}
