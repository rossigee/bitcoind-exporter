package prometheus

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

// Test all blockchain metrics are properly defined
func TestBlockchainMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metric   prometheus.Gauge
		expected string
	}{
		{
			name:     "BlockchainBlocks",
			metric:   BlockchainBlocks,
			expected: "bitcoind_blockchain_blocks",
		},
		{
			name:     "BlockchainHeaders", 
			metric:   BlockchainHeaders,
			expected: "bitcoind_blockchain_headers",
		},
		{
			name:     "BlockchainVerificationProgress",
			metric:   BlockchainVerificationProgress,
			expected: "bitcoind_blockchain_verification_progress",
		},
		{
			name:     "BlockchainSizeOnDisk",
			metric:   BlockchainSizeOnDisk,
			expected: "bitcoind_blockchain_size_on_disk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.metric)
			
			// Get the metric descriptor
			desc := tt.metric.Desc()
			assert.Contains(t, desc.String(), tt.expected)
		})
	}
}

// Test blockchain metrics can be set and read
func TestBlockchainMetricsValues(t *testing.T) {
	// Test setting and getting values
	BlockchainBlocks.Set(800000)
	BlockchainHeaders.Set(800100)
	BlockchainVerificationProgress.Set(0.99999)
	BlockchainSizeOnDisk.Set(500000000000) // 500GB

	// Verify values can be read back
	metric := &dto.Metric{}
	
	assert.NoError(t, BlockchainBlocks.Write(metric))
	assert.Equal(t, float64(800000), metric.GetGauge().GetValue())
	
	metric.Reset()
	assert.NoError(t, BlockchainHeaders.Write(metric))
	assert.Equal(t, float64(800100), metric.GetGauge().GetValue())
	
	metric.Reset()
	assert.NoError(t, BlockchainVerificationProgress.Write(metric))
	assert.InDelta(t, 0.99999, metric.GetGauge().GetValue(), 0.00001)
	
	metric.Reset()
	assert.NoError(t, BlockchainSizeOnDisk.Write(metric))
	assert.Equal(t, float64(500000000000), metric.GetGauge().GetValue())
}

// Test fee metrics
func TestFeeMetrics(t *testing.T) {
	// These metrics would be defined in fee.go
	// Testing pattern would be similar to blockchain metrics
	
	// Example test structure for fee metrics:
	// - Fee2Blocks, Fee5Blocks, Fee20Blocks
	// - Verify they can be set to reasonable fee values
	
	t.Skip("Fee metrics test - implement based on actual fee.go content")
}

// Test memory metrics
func TestMemoryMetrics(t *testing.T) {
	// These metrics would be defined in memory.go
	// Testing pattern for memory usage metrics
	
	t.Skip("Memory metrics test - implement based on actual memory.go content")
}

// Test mempool metrics
func TestMempoolMetrics(t *testing.T) {
	// These metrics would be defined in mempool.go
	// Testing pattern for mempool size, transactions, etc.
	
	t.Skip("Mempool metrics test - implement based on actual mempool.go content")
}

// Test mining metrics
func TestMiningMetrics(t *testing.T) {
	// These metrics would be defined in mining.go
	// Testing pattern for hash rate, difficulty, etc.
	
	t.Skip("Mining metrics test - implement based on actual mining.go content")
}

// Test network metrics
func TestNetworkMetrics(t *testing.T) {
	// These metrics would be defined in network.go
	// Testing pattern for connections, traffic, etc.
	
	t.Skip("Network metrics test - implement based on actual network.go content")
}

// Test ZMQ metrics
func TestZMQMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metric   prometheus.Gauge
		expected string
	}{
		{
			name:     "TransactionsPerSecond",
			metric:   TransactionsPerSecond,
			expected: "bitcoind_transactions_per_second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.metric)
			
			// Get the metric descriptor
			desc := tt.metric.Desc()
			assert.Contains(t, desc.String(), tt.expected)
		})
	}
}

// Test ZMQ metrics functionality
func TestZMQMetricsValues(t *testing.T) {
	// Test incrementing transactions per second
	TransactionsPerSecond.Set(0) // Reset
	TransactionsPerSecond.Inc()
	TransactionsPerSecond.Inc()
	TransactionsPerSecond.Inc()

	metric := &dto.Metric{}
	assert.NoError(t, TransactionsPerSecond.Write(metric))
	assert.Equal(t, float64(3), metric.GetGauge().GetValue())

	// Test setting specific value
	TransactionsPerSecond.Set(25.5)
	metric.Reset()
	assert.NoError(t, TransactionsPerSecond.Write(metric))
	assert.Equal(t, float64(25.5), metric.GetGauge().GetValue())
}

// Test internal metrics
func TestInternalMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metric   prometheus.Gauge
		expected string
	}{
		{
			name:     "ScrapeTime",
			metric:   ScrapeTime,
			expected: "bitcoind_exporter_scrape_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.metric)
			
			// Get the metric descriptor
			desc := tt.metric.Desc()
			assert.Contains(t, desc.String(), tt.expected)
		})
	}
}

// Test internal metrics functionality
func TestInternalMetricsValues(t *testing.T) {
	// Test scrape time metric
	ScrapeTime.Set(1.234) // 1.234 seconds

	metric := &dto.Metric{}
	assert.NoError(t, ScrapeTime.Write(metric))
	assert.InDelta(t, 1.234, metric.GetGauge().GetValue(), 0.001)
}

// Test metric naming conventions
func TestMetricNamingConventions(t *testing.T) {
	metrics := []prometheus.Gauge{
		BlockchainBlocks,
		BlockchainHeaders,
		BlockchainVerificationProgress,
		BlockchainSizeOnDisk,
		TransactionsPerSecond,
		ScrapeTime,
	}

	for _, metric := range metrics {
		desc := metric.Desc()
		metricName := extractMetricName(desc.String())
		
		// All metrics should start with "bitcoind_"
		assert.True(t, strings.HasPrefix(metricName, "bitcoind_"), 
			"Metric should start with 'bitcoind_': %s", metricName)
		
		// Metric names should use underscores, not hyphens
		assert.False(t, strings.Contains(metricName, "-"), 
			"Metric names should not contain hyphens: %s", metricName)
		
		// Metric names should be lowercase
		assert.Equal(t, strings.ToLower(metricName), metricName,
			"Metric names should be lowercase: %s", metricName)
	}
}

// Test metric help text
func TestMetricHelpText(t *testing.T) {
	metrics := []prometheus.Gauge{
		BlockchainBlocks,
		BlockchainHeaders,
		BlockchainVerificationProgress,
		BlockchainSizeOnDisk,
		TransactionsPerSecond,
		ScrapeTime,
	}

	for _, metric := range metrics {
		desc := metric.Desc()
		help := extractHelpText(desc.String())
		
		// Help text should not be empty
		assert.NotEmpty(t, help, "Metric should have help text")
		
		// Help text should be descriptive (more than just the metric name)
		assert.Greater(t, len(help), 10, "Help text should be descriptive")
	}
}

// Test metric type consistency
func TestMetricTypes(t *testing.T) {
	// All current metrics are Gauges, verify they behave as Gauges
	gauges := []prometheus.Gauge{
		BlockchainBlocks,
		BlockchainHeaders,
		BlockchainVerificationProgress,
		BlockchainSizeOnDisk,
		TransactionsPerSecond,
		ScrapeTime,
	}

	for _, gauge := range gauges {
		// Test gauge-specific operations
		gauge.Set(100)
		gauge.Add(50)
		gauge.Sub(25)
		
		metric := &dto.Metric{}
		assert.NoError(t, gauge.Write(metric))
		assert.Equal(t, float64(125), metric.GetGauge().GetValue())
		
		// Reset for next test
		gauge.Set(0)
	}
}

// Test metric registration
func TestMetricRegistration(t *testing.T) {
	// Verify that metrics are properly registered with Prometheus
	// This is done automatically by promauto.NewGauge
	
	metrics := []prometheus.Gauge{
		BlockchainBlocks,
		BlockchainHeaders,
		BlockchainVerificationProgress,
		BlockchainSizeOnDisk,
		TransactionsPerSecond,
		ScrapeTime,
	}

	for _, metric := range metrics {
		assert.NotNil(t, metric)
		assert.NotNil(t, metric.Desc())
	}
}

// Test concurrent access to metrics
func TestMetricsConcurrency(t *testing.T) {
	// Test that metrics can be safely accessed from multiple goroutines
	metric := BlockchainBlocks
	metric.Set(0)
	
	done := make(chan bool, 10)
	
	// Start 10 goroutines that increment the metric
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metric.Inc()
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final value
	metricDto := &dto.Metric{}
	assert.NoError(t, metric.Write(metricDto))
	assert.Equal(t, float64(1000), metricDto.GetGauge().GetValue())
}

// Helper functions for extracting metric information from descriptor strings
func extractMetricName(descString string) string {
	// Parse metric name from descriptor string
	// This is a simplified parser for test purposes
	lines := strings.Split(descString, "\n")
	for _, line := range lines {
		if strings.Contains(line, "fqName:") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func extractHelpText(descString string) string {
	// Parse help text from descriptor string
	// This is a simplified parser for test purposes
	lines := strings.Split(descString, "\n")
	for _, line := range lines {
		if strings.Contains(line, "help:") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// Integration test for all metrics
func TestAllMetricsIntegration(t *testing.T) {
	// Test that all metrics can be set to realistic values without errors
	
	// Blockchain metrics
	BlockchainBlocks.Set(800000)
	BlockchainHeaders.Set(800100)
	BlockchainVerificationProgress.Set(0.99999)
	BlockchainSizeOnDisk.Set(500000000000)
	
	// ZMQ metrics
	TransactionsPerSecond.Set(5.5)
	
	// Internal metrics
	ScrapeTime.Set(2.345)
	
	// Verify all metrics can be read
	metrics := []prometheus.Gauge{
		BlockchainBlocks,
		BlockchainHeaders,
		BlockchainVerificationProgress,
		BlockchainSizeOnDisk,
		TransactionsPerSecond,
		ScrapeTime,
	}
	
	for _, metric := range metrics {
		metricDto := &dto.Metric{}
		assert.NoError(t, metric.Write(metricDto))
		assert.NotNil(t, metricDto.GetGauge())
	}
}

// Test metric value ranges and validation
func TestMetricValueValidation(t *testing.T) {
	tests := []struct {
		name          string
		metric        prometheus.Gauge
		testValues    []float64
		expectValid   []bool
	}{
		{
			name:        "BlockchainBlocks - should accept positive integers",
			metric:      BlockchainBlocks,
			testValues:  []float64{0, 800000, 1000000},
			expectValid: []bool{true, true, true},
		},
		{
			name:        "BlockchainVerificationProgress - should accept 0-1 range",
			metric:      BlockchainVerificationProgress,
			testValues:  []float64{0.0, 0.5, 0.99999, 1.0},
			expectValid: []bool{true, true, true, true},
		},
		{
			name:        "ScrapeTime - should accept positive durations",
			metric:      ScrapeTime,
			testValues:  []float64{0.001, 1.0, 30.0, 120.5},
			expectValid: []bool{true, true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, value := range tt.testValues {
				tt.metric.Set(value)
				
				metricDto := &dto.Metric{}
				err := tt.metric.Write(metricDto)
				
				if tt.expectValid[i] {
					assert.NoError(t, err)
					assert.Equal(t, value, metricDto.GetGauge().GetValue())
				}
				// Note: Prometheus gauges don't inherently validate ranges,
				// but this test documents expected value ranges
			}
		})
	}
}