package registry

import (
	"sort"

	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
)

// Store is a generic, named collection of resources. One Store instance
// serves each resource type (servers, skills, hooks, permissions, templates,
// prompts). The PT constraint ensures T's pointer type implements Resource.
type Store[T any, PT interface {
	model.Resource
	*T
}] struct {
	items    map[string]T
	kind     string
	sortFunc func(a, b T) int
}

// NewStore creates an empty Store for the given resource kind.
func NewStore[T any, PT interface {
	model.Resource
	*T
}](kind string) *Store[T, PT] {
	return &Store[T, PT]{
		items: make(map[string]T),
		kind:  kind,
	}
}

// WithSort returns the store with a custom sort function set.
// The sort function should return a negative number when a < b,
// zero when a == b, and a positive number when a > b.
func (s *Store[T, PT]) WithSort(fn func(a, b T) int) *Store[T, PT] {
	s.sortFunc = fn
	return s
}

// Kind returns the resource kind name (e.g. "server", "skill").
func (s *Store[T, PT]) Kind() string {
	return s.kind
}

// Add inserts a new resource. Returns an error if the name already exists.
func (s *Store[T, PT]) Add(item T) error {
	p := PT(&item)
	name := p.ResourceName()
	if name == "" {
		return &hysterr.ValidationError{Field: "name", Message: s.kind + " name must not be empty"}
	}
	if _, exists := s.items[name]; exists {
		return &hysterr.AlreadyExists{Kind: s.kind, Name: name}
	}
	s.items[name] = item
	return nil
}

// Get retrieves a resource by name. Returns the value and whether it was found.
func (s *Store[T, PT]) Get(name string) (T, bool) {
	item, ok := s.items[name]
	return item, ok
}

// Update replaces an existing resource. Returns an error if not found.
func (s *Store[T, PT]) Update(item T) error {
	p := PT(&item)
	name := p.ResourceName()
	if _, exists := s.items[name]; !exists {
		return &hysterr.ResourceNotFound{Kind: s.kind, Name: name}
	}
	s.items[name] = item
	return nil
}

// Delete removes a resource by name. Returns an error if not found.
func (s *Store[T, PT]) Delete(name string) error {
	if _, exists := s.items[name]; !exists {
		return &hysterr.ResourceNotFound{Kind: s.kind, Name: name}
	}
	delete(s.items, name)
	return nil
}

// List returns all resources in sorted order (by name, or by custom sort).
func (s *Store[T, PT]) List() []T {
	result := make([]T, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	if s.sortFunc != nil {
		sort.Slice(result, func(i, j int) bool {
			return s.sortFunc(result[i], result[j]) < 0
		})
	} else {
		sort.Slice(result, func(i, j int) bool {
			pi := PT(&result[i])
			pj := PT(&result[j])
			return pi.ResourceName() < pj.ResourceName()
		})
	}
	return result
}

// Items returns a shallow copy of all resources as a map.
func (s *Store[T, PT]) Items() map[string]T {
	cp := make(map[string]T, len(s.items))
	for k, v := range s.items {
		cp[k] = v
	}
	return cp
}

// SetItems replaces all resources. Each item's name is derived from the
// Resource interface and used as the map key.
func (s *Store[T, PT]) SetItems(items map[string]T) {
	s.items = make(map[string]T, len(items))
	for k, v := range items {
		p := PT(&v)
		p.SetResourceName(k)
		s.items[k] = v
	}
}

// Len returns the number of resources in the store.
func (s *Store[T, PT]) Len() int {
	return len(s.items)
}
