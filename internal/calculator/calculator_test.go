package calculator

import (
	"math"
	"testing"
)

const floatEpsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatEpsilon
}

func TestCalculate_SonnetKnownInput(t *testing.T) {
	// Sonnet 4 family rates: input 3.00, output 15.00, cache read 0.30, cache write 3.75 per MTok.
	usage := TokenUsage{
		InputTokens:      1_000_000,
		OutputTokens:     500_000,
		CacheReadTokens:  2_000_000,
		CacheWriteTokens: 100_000,
	}
	got := Calculate("claude-sonnet-4-6", usage)

	wantInput := 1.0 * 3.00      // 3.00
	wantOutput := 0.5 * 15.00    // 7.50
	wantCacheRead := 2.0 * 0.30  // 0.60
	wantCacheWrite := 0.1 * 3.75 // 0.375
	wantTotal := wantInput + wantOutput + wantCacheRead + wantCacheWrite

	if !almostEqual(got.InputCost, wantInput) {
		t.Errorf("InputCost = %v, want %v", got.InputCost, wantInput)
	}
	if !almostEqual(got.OutputCost, wantOutput) {
		t.Errorf("OutputCost = %v, want %v", got.OutputCost, wantOutput)
	}
	if !almostEqual(got.CacheReadCost, wantCacheRead) {
		t.Errorf("CacheReadCost = %v, want %v", got.CacheReadCost, wantCacheRead)
	}
	if !almostEqual(got.CacheWriteCost, wantCacheWrite) {
		t.Errorf("CacheWriteCost = %v, want %v", got.CacheWriteCost, wantCacheWrite)
	}
	if !almostEqual(got.TotalCost, wantTotal) {
		t.Errorf("TotalCost = %v, want %v", got.TotalCost, wantTotal)
	}
}

func TestCalculate_ZeroUsageIsZeroCost(t *testing.T) {
	got := Calculate("claude-sonnet-4-6", TokenUsage{})
	if got.InputCost != 0 || got.OutputCost != 0 || got.CacheReadCost != 0 || got.CacheWriteCost != 0 || got.TotalCost != 0 {
		t.Errorf("expected all-zero CostBreakdown, got %+v", got)
	}
}

func TestTokenUsage_Total(t *testing.T) {
	tests := []struct {
		name  string
		usage TokenUsage
		want  int64
	}{
		{"zero", TokenUsage{}, 0},
		{"all-fields", TokenUsage{InputTokens: 1, OutputTokens: 2, CacheReadTokens: 4, CacheWriteTokens: 8}, 15},
		{"large", TokenUsage{InputTokens: 1_000_000, OutputTokens: 2_000_000, CacheReadTokens: 3_000_000, CacheWriteTokens: 4_000_000}, 10_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.usage.Total(); got != tt.want {
				t.Errorf("Total() = %d, want %d", got, tt.want)
			}
		})
	}
}
