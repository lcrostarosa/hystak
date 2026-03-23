package model

// Resource is the constraint interface for all registry resource types.
// Every domain type that lives in a Store must implement it.
type Resource interface {
	ResourceName() string
	SetResourceName(name string)
}
