package model

import (
	"maps"
	"slices"
)

// slicesEqualNil compares two slices treating nil and empty as equivalent.
func slicesEqualNil[T comparable](a, b []T) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return slices.Equal(a, b)
}

// mapsEqualNil compares two maps treating nil and empty as equivalent.
func mapsEqualNil[K, V comparable](a, b map[K]V) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return maps.Equal(a, b)
}
