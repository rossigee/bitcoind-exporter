package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertBTCkBToSatVb(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "standard fee rate",
			input:    0.00001000, // 1000 sats/kB
			expected: 1.0,        // 1 sat/vB
		},
		{
			name:     "high fee rate",
			input:    0.00010000, // 10000 sats/kB
			expected: 10.0,       // 10 sats/vB
		},
		{
			name:     "low fee rate",
			input:    0.00000100, // 100 sats/kB
			expected: 0.1,        // 0.1 sats/vB
		},
		{
			name:     "zero fee rate",
			input:    0.0,
			expected: 0.0,
		},
		{
			name:     "very high fee rate",
			input:    0.00100000, // 100000 sats/kB
			expected: 100.0,      // 100 sats/vB
		},
		{
			name:     "fractional result",
			input:    0.00000250, // 250 sats/kB
			expected: 0.25,       // 0.25 sats/vB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertBTCkBToSatVb(tt.input)
			assert.InDelta(t, tt.expected, result, 0.000001) // Allow small floating point differences
		})
	}
}

func TestConvertBTCkBToSatVb_EdgeCases(t *testing.T) {
	t.Run("negative input", func(t *testing.T) {
		result := ConvertBTCkBToSatVb(-0.00001000)
		assert.Equal(t, -1.0, result)
	})

	t.Run("very small positive", func(t *testing.T) {
		result := ConvertBTCkBToSatVb(0.00000001) // 1 sat/kB
		expected := 0.001                         // 0.001 sats/vB
		assert.InDelta(t, expected, result, 0.000001)
	})

	t.Run("infinity", func(t *testing.T) {
		result := ConvertBTCkBToSatVb(math.Inf(1))
		assert.True(t, math.IsInf(result, 1))
	})

	t.Run("NaN", func(t *testing.T) {
		result := ConvertBTCkBToSatVb(math.NaN())
		assert.True(t, math.IsNaN(result))
	})
}

// Test the mathematical constants and formula
func TestConvertBTCkBToSatVb_Formula(t *testing.T) {
	// Test that the conversion formula is correct
	btcPerKB := 0.00001000 // 1000 sats/kB input

	// Manual calculation:
	// 1. Convert kB to bytes: 0.00001000 / 1000 = 0.00000001 BTC/byte
	// 2. Convert BTC to satoshis: 0.00000001 * 100000000 = 1 sat/byte
	expectedSatPerByte := (btcPerKB / 1000.0) * 100000000.0

	result := ConvertBTCkBToSatVb(btcPerKB)

	assert.Equal(t, expectedSatPerByte, result)
	assert.Equal(t, 1.0, result)
}

// Benchmark the conversion function
func BenchmarkConvertBTCkBToSatVb(b *testing.B) {
	input := 0.00001000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertBTCkBToSatVb(input)
	}
}

// Property-based test: conversion should be reversible
func TestConvertBTCkBToSatVb_Reversible(t *testing.T) {
	inputs := []float64{
		0.00001000,
		0.00010000,
		0.00000100,
		0.00050000,
	}

	for _, input := range inputs {
		t.Run("reversible", func(t *testing.T) {
			satPerVb := ConvertBTCkBToSatVb(input)

			// Reverse calculation: sat/vB back to BTC/kB
			// satPerVb / 100000000 * 1000 = btcPerKB
			reversed := (satPerVb / 100000000.0) * 1000.0

			assert.InDelta(t, input, reversed, 0.000000001)
		})
	}
}
