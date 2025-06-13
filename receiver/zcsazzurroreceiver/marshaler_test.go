package zcsazzurroreceiver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/zmoog/zcs/azzurro"
)

func TestAzzurroRealtimeDataMarshaler_UnmarshalMetrics(t *testing.T) {
	// Load test data
	testDataPath := "testdata/response.json"
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	var response azzurro.RealtimeDataResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err, "Failed to unmarshal test data")

	// Create marshaler
	logger := zap.NewNop()
	marshaler := &azzurroRealtimeDataMarshaler{logger: logger}

	// Test successful response
	t.Run("successful response", func(t *testing.T) {
		// Extract the first device from the response
		deviceKey := "my-serial-number"
		deviceMetrics := response.RealtimeData.Params.Value[0][deviceKey]

		metrics, err := marshaler.UnmarshalMetrics(deviceKey, deviceMetrics)
		require.NoError(t, err)

		// Verify metrics structure
		assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Should have 1 resource metric")

		resourceMetrics := metrics.ResourceMetrics().At(0)

		// Verify resource attributes
		resource := resourceMetrics.Resource()
		thingKey, exists := resource.Attributes().Get("thing_key")
		assert.True(t, exists, "thing_key attribute should exist")
		assert.Equal(t, "my-serial-number", thingKey.Str())

		// Verify scope metrics
		assert.Equal(t, 1, resourceMetrics.ScopeMetrics().Len(), "Should have 1 scope metric")
		scopeMetrics := resourceMetrics.ScopeMetrics().At(0)

		scope := scopeMetrics.Scope()
		assert.Equal(t, scopeName, scope.Name())
		assert.Equal(t, scopeVersion, scope.Version())

		// Expected metrics count: 7 power + 2 battery (soc + total) + 7 energy (daily) + 7 energy (total) = 23 metrics
		expectedMetricsCount := 23
		assert.Equal(t, expectedMetricsCount, scopeMetrics.Metrics().Len(), "Should have %d metrics", expectedMetricsCount)

		// Verify specific metrics
		metricNames := make(map[string]bool)
		for i := 0; i < scopeMetrics.Metrics().Len(); i++ {
			metric := scopeMetrics.Metrics().At(i)
			metricNames[metric.Name()] = true

			// Verify timestamp is set
			switch metric.Type() {
			case pmetric.MetricTypeGauge:
				assert.Greater(t, metric.Gauge().DataPoints().Len(), 0, "Gauge should have data points")
				dp := metric.Gauge().DataPoints().At(0)
				assert.Greater(t, dp.Timestamp(), pcommon.Timestamp(0), "Timestamp should be set")
			case pmetric.MetricTypeSum:
				assert.Greater(t, metric.Sum().DataPoints().Len(), 0, "Sum should have data points")
				dp := metric.Sum().DataPoints().At(0)
				assert.Greater(t, dp.Timestamp(), pcommon.Timestamp(0), "Timestamp should be set")
				assert.Greater(t, dp.StartTimestamp(), pcommon.Timestamp(0), "Start timestamp should be set")
				assert.True(t, metric.Sum().IsMonotonic(), "Sum should be monotonic")
				assert.Equal(t, pmetric.AggregationTemporalityCumulative, metric.Sum().AggregationTemporality())
			}
		}

		// Verify expected power metrics (gauges)
		powerMetrics := []string{
			"power_autoconsuming", "power_charging", "power_consuming",
			"power_discharging", "power_exporting", "power_generating", "power_importing",
		}
		for _, name := range powerMetrics {
			assert.True(t, metricNames[name], "Power metric %s should exist", name)
		}

		// Verify battery metrics
		assert.True(t, metricNames["battery_soc"], "Battery SOC metric should exist")
		assert.True(t, metricNames["battery_cycletime_total"], "Battery cycletime total metric should exist")

		// Verify energy metrics (daily sums)
		energyMetrics := []string{
			"energy_autoconsuming", "energy_charging", "energy_consuming",
			"energy_discharging", "energy_exporting", "energy_generating", "energy_importing",
		}
		for _, name := range energyMetrics {
			assert.True(t, metricNames[name], "Energy metric %s should exist", name)
		}

		// Verify total energy metrics (lifetime sums)
		energyTotalMetrics := []string{
			"energy_autoconsuming_total", "energy_charging_total", "energy_consuming_total",
			"energy_discharging_total", "energy_exporting_total", "energy_generating_total", "energy_importing_total",
		}
		for _, name := range energyTotalMetrics {
			assert.True(t, metricNames[name], "Energy total metric %s should exist", name)
		}
	})

	// Test empty response
	t.Run("empty response", func(t *testing.T) {
		// Create empty metrics
		emptyMetrics := azzurro.InverterMetrics{}

		metrics, err := marshaler.UnmarshalMetrics("test-device", emptyMetrics)
		require.NoError(t, err)
		// Should still create metrics even with empty data
		assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Should have 1 resource metric even for empty data")
	})
}

func TestAzzurroRealtimeDataMarshaler_MetricValues(t *testing.T) {
	// Load test data
	testDataPath := "testdata/response.json"
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err)

	var response azzurro.RealtimeDataResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err)

	// Create marshaler
	logger := zap.NewNop()
	marshaler := &azzurroRealtimeDataMarshaler{logger: logger}

	// Extract the first device from the response
	deviceKey := "my-serial-number"
	deviceMetrics := response.RealtimeData.Params.Value[0][deviceKey]

	metrics, err := marshaler.UnmarshalMetrics(deviceKey, deviceMetrics)
	require.NoError(t, err)

	resourceMetrics := metrics.ResourceMetrics().At(0)
	scopeMetrics := resourceMetrics.ScopeMetrics().At(0)

	// Create a map of metric name to value for easier testing
	metricValues := make(map[string]interface{})
	for i := 0; i < scopeMetrics.Metrics().Len(); i++ {
		metric := scopeMetrics.Metrics().At(i)
		name := metric.Name()

		switch metric.Type() {
		case pmetric.MetricTypeGauge:
			dp := metric.Gauge().DataPoints().At(0)
			if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
				metricValues[name] = dp.IntValue()
			} else {
				metricValues[name] = dp.DoubleValue()
			}
		case pmetric.MetricTypeSum:
			dp := metric.Sum().DataPoints().At(0)
			if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
				metricValues[name] = dp.IntValue()
			} else {
				metricValues[name] = dp.DoubleValue()
			}
		}
	}

	// Verify specific values from test data
	assert.Equal(t, float64(950), metricValues["power_importing"])
	assert.Equal(t, float64(950), metricValues["power_consuming"])
	assert.Equal(t, float64(0), metricValues["power_generating"])
	assert.Equal(t, int64(20), metricValues["battery_soc"])
	assert.Equal(t, float64(27.4), metricValues["energy_consuming"])
	assert.Equal(t, float64(20.46), metricValues["energy_generating"])
}

func TestAzzurroRealtimeDataMarshaler_Timestamp(t *testing.T) {
	// Create test metrics with known timestamp
	testTime := time.Date(2024, 10, 22, 19, 46, 52, 0, time.UTC)
	testMetrics := azzurro.InverterMetrics{
		LastUpdate:          testTime,
		PowerImporting:      100.0,
		BatterySoC:          50,
		EnergyAutoconsuming: 10.5,
		ThingFind:           "2024-06-04T08:55:36Z",
	}

	logger := zap.NewNop()
	marshaler := &azzurroRealtimeDataMarshaler{logger: logger}

	metrics, err := marshaler.UnmarshalMetrics("test-device", testMetrics)
	require.NoError(t, err)

	resourceMetrics := metrics.ResourceMetrics().At(0)
	scopeMetrics := resourceMetrics.ScopeMetrics().At(0)

	// Check that all metrics have the correct timestamp
	expectedTimestamp := pcommon.Timestamp(testTime.UnixNano())
	expectedDailyStartTimestamp := pcommon.Timestamp(testTime.Truncate(24 * time.Hour).UnixNano())

	// Parse thingFind for total metrics start timestamp
	thingFindTime, err := time.Parse("2006-01-02T15:04:05Z", "2024-06-04T08:55:36Z")
	require.NoError(t, err)
	expectedThingFindTimestamp := pcommon.Timestamp(thingFindTime.UnixNano())

	for i := 0; i < scopeMetrics.Metrics().Len(); i++ {
		metric := scopeMetrics.Metrics().At(i)

		switch metric.Type() {
		case pmetric.MetricTypeGauge:
			dp := metric.Gauge().DataPoints().At(0)
			assert.Equal(t, expectedTimestamp, dp.Timestamp(), "Gauge metric %s should have correct timestamp", metric.Name())
		case pmetric.MetricTypeSum:
			dp := metric.Sum().DataPoints().At(0)
			assert.Equal(t, expectedTimestamp, dp.Timestamp(), "Sum metric %s should have correct timestamp", metric.Name())

			// Different start timestamps based on metric type
			if strings.HasSuffix(metric.Name(), "_total") {
				// Total metrics use thingFind timestamp as start
				assert.Equal(t, expectedThingFindTimestamp, dp.StartTimestamp(), "Sum metric %s should have thingFind start timestamp", metric.Name())
			} else {
				// Daily metrics use start of day as start timestamp
				assert.Equal(t, expectedDailyStartTimestamp, dp.StartTimestamp(), "Sum metric %s should have daily start timestamp", metric.Name())
			}
		}
	}
}
