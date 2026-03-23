package model

// DriftStatus describes the sync state of a deployed resource.
type DriftStatus string

const (
	DriftSynced    DriftStatus = "synced"
	DriftDrifted   DriftStatus = "drifted"
	DriftMissing   DriftStatus = "missing"
	DriftUnmanaged DriftStatus = "unmanaged"
)

// Valid reports whether s is a known drift status.
func (s DriftStatus) Valid() bool {
	switch s {
	case DriftSynced, DriftDrifted, DriftMissing, DriftUnmanaged:
		return true
	}
	return false
}
