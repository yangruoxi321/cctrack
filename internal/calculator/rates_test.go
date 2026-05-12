package calculator

import "testing"

func TestGetRates(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantFamily string
		wantInput  float64
	}{
		{
			name:       "opus 4.7 dated model",
			model:      "claude-opus-4-7-20251023",
			wantFamily: "claude-opus-4-7",
			wantInput:  5.00,
		},
		{
			name:       "legacy opus-4 (bare)",
			model:      "claude-opus-4",
			wantFamily: "claude-opus-4",
			wantInput:  15.00,
		},
		{
			name:       "legacy opus-4.1 dated model",
			model:      "claude-opus-4-1-20250805",
			wantFamily: "claude-opus-4",
			wantInput:  15.00,
		},
		{
			name:       "sonnet 4.6",
			model:      "claude-sonnet-4-6",
			wantFamily: "claude-sonnet-4",
			wantInput:  3.00,
		},
		{
			name:       "haiku 4.5",
			model:      "claude-haiku-4-5",
			wantFamily: "claude-haiku-4-5",
			wantInput:  1.00,
		},
		{
			name:       "unknown model falls back to sonnet-4",
			model:      "unknown-model-xyz",
			wantFamily: "claude-sonnet-4",
			wantInput:  3.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRates(tt.model)
			if got == nil {
				t.Fatalf("GetRates(%q) returned nil", tt.model)
			}
			if got.Family != tt.wantFamily {
				t.Errorf("GetRates(%q).Family = %q, want %q", tt.model, got.Family, tt.wantFamily)
			}
			if got.InputPerMToken != tt.wantInput {
				t.Errorf("GetRates(%q).InputPerMToken = %v, want %v", tt.model, got.InputPerMToken, tt.wantInput)
			}
		})
	}
}
