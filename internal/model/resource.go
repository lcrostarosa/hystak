package model

// Resource is the constraint satisfied by all registry resource types.
// Every resource has a name (the map key in YAML) that can be get/set.
type Resource interface {
	ResourceName() string
	SetResourceName(name string)
}
