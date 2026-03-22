package tui

import "testing"

func TestConflictDescriptions_AllTypesPresent(t *testing.T) {
	expected := []string{"skill", "hook", "permission", "claude_md"}
	for _, rt := range expected {
		fn, ok := conflictDescriptions[rt]
		if !ok {
			t.Errorf("missing conflict description for %q", rt)
			continue
		}
		result := fn("test-name")
		if result == "" {
			t.Errorf("conflictDescriptions[%q] returned empty string", rt)
		}
	}
}

func TestConflictDescriptions_UnknownTypeFallback(t *testing.T) {
	// The View method handles unknown types with a generic fallback.
	// Verify the map doesn't have unexpected entries.
	if len(conflictDescriptions) != 4 {
		t.Errorf("expected 4 entries in conflictDescriptions, got %d", len(conflictDescriptions))
	}
}
