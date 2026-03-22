package model

import "testing"

func TestDriftStatus_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value DriftStatus
		want  bool
	}{
		{"synced", DriftSynced, true},
		{"drifted", DriftDrifted, true},
		{"missing", DriftMissing, true},
		{"unmanaged", DriftUnmanaged, true},
		{"empty", DriftStatus(""), false},
		{"unknown", DriftStatus("stale"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("DriftStatus(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
