package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected float64
	}{
		{
			name:     "true returns 1.0",
			input:    true,
			expected: 1.0,
		},
		{
			name:     "false returns 0.0",
			input:    false,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BoolToFloat64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringToBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello:world",
			expected: "aGVsbG86d29ybGQ=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "username:password format",
			input:    "bitcoin:secretpassword123",
			expected: "Yml0Y29pbjpzZWNyZXRwYXNzd29yZDEyMw==",
		},
		{
			name:     "special characters",
			input:    "user@host:pass!@#$",
			expected: "dXNlckBob3N0OnBhc3MhQCMk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToBase64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnyNil(t *testing.T) {
	tests := []struct {
		name     string
		params   []interface{}
		expected bool
	}{
		{
			name:     "no nil values",
			params:   []interface{}{"hello", 123, true},
			expected: false,
		},
		{
			name:     "contains nil",
			params:   []interface{}{"hello", nil, true},
			expected: true,
		},
		{
			name:     "all nil",
			params:   []interface{}{nil, nil, nil},
			expected: true,
		},
		{
			name:     "empty slice",
			params:   []interface{}{},
			expected: false,
		},
		{
			name:     "nil pointer",
			params:   []interface{}{(*string)(nil)},
			expected: true,
		},
		{
			name:     "nil slice",
			params:   []interface{}{([]string)(nil)},
			expected: true,
		},
		{
			name:     "nil map",
			params:   []interface{}{(map[string]string)(nil)},
			expected: true,
		},
		{
			name:     "valid pointer",
			params:   []interface{}{&[]string{"test"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnyNil(tt.params...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkBoolToFloat64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BoolToFloat64(i%2 == 0)
	}
}

func BenchmarkStringToBase64(b *testing.B) {
	input := "username:password123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		StringToBase64(input)
	}
}

func BenchmarkAnyNil(b *testing.B) {
	params := []interface{}{"hello", 123, true, "world", 456}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AnyNil(params...)
	}
}
