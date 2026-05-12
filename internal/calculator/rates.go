package calculator

import "strings"

type ModelRates struct {
	Family              string
	InputPerMToken      float64
	OutputPerMToken     float64
	CacheReadPerMToken  float64
	CacheWritePerMToken float64
}

// Rates maps model family prefixes to their pricing.
// Models are matched by prefix in order: more specific prefixes MUST come before
// broader ones (e.g. "claude-opus-4-7" before "claude-opus-4").
// Cache write prices are the 5-minute tier — cctrack's parser does not split
// 5m vs 1h cache writes (see internal/parser/parser.go), so this is the
// project-wide assumption.
var Rates = []ModelRates{
	// ===== Opus =====
	// Opus 4.5 / 4.6 / 4.7 share the new (Apr 2026) pricing.
	{
		Family:              "claude-opus-4-7",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-6",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-5",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	// Opus 4 / 4.1 keep legacy pricing — this prefix catches both.
	{
		Family:              "claude-opus-4",
		InputPerMToken:      15.00,
		OutputPerMToken:     75.00,
		CacheReadPerMToken:  1.50,
		CacheWritePerMToken: 18.75,
	},
	// Opus 3 (deprecated) — same legacy pricing.
	{
		Family:              "claude-opus-3",
		InputPerMToken:      15.00,
		OutputPerMToken:     75.00,
		CacheReadPerMToken:  1.50,
		CacheWritePerMToken: 18.75,
	},

	// ===== Sonnet =====
	// Sonnet 3.7 / 4 / 4.5 / 4.6 all share the same pricing.
	{
		Family:              "claude-sonnet-4",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-sonnet-3-7",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},

	// ===== Haiku =====
	// Haiku 4.5 — new pricing.
	{
		Family:              "claude-haiku-4-5",
		InputPerMToken:      1.00,
		OutputPerMToken:     5.00,
		CacheReadPerMToken:  0.10,
		CacheWritePerMToken: 1.25,
	},
	// Haiku 3.5.
	{
		Family:              "claude-haiku-3-5",
		InputPerMToken:      0.80,
		OutputPerMToken:     4.00,
		CacheReadPerMToken:  0.08,
		CacheWritePerMToken: 1.00,
	},
	// Haiku 3.
	{
		Family:              "claude-haiku-3",
		InputPerMToken:      0.25,
		OutputPerMToken:     1.25,
		CacheReadPerMToken:  0.03,
		CacheWritePerMToken: 0.30,
	},
}

func GetRates(model string) *ModelRates {
	for i := range Rates {
		if strings.HasPrefix(model, Rates[i].Family) {
			return &Rates[i]
		}
	}
	// Fallback for unknown models — default to sonnet-4 rates as the most common.
	// Index 5 = "claude-sonnet-4" entry above.
	return &Rates[5]
}
