package registry

import (
	"sort"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
)

// Store is a generic named-resource collection with CRUD operations.
// T is the value type (e.g. model.ServerDef); PT is *T and must satisfy Resource.
type Store[T any, PT interface {
	model.Resource
	*T
}] struct {
	items    map[string]T
	kind     string               // "server", "skill", etc. — for error messages
	sortFunc func(a, b T) bool    // optional custom sort; nil means sort by name
}

// NewStore creates an empty Store for resources of the given kind.
func NewStore[T any, PT interface {
	model.Resource
	*T
}](kind string) *Store[T, PT] {
	return &Store[T, PT]{
		items: make(map[string]T),
		kind:  kind,
	}
}

// WithSort sets a custom sort function for List output.
func (s *Store[T, PT]) WithSort(fn func(a, b T) bool) *Store[T, PT] {
	s.sortFunc = fn
	return s
}

// Add inserts a resource. Returns an error if the name already exists.
func (s *Store[T, PT]) Add(item T) error {
	p := PT(&item)
	name := p.ResourceName()
	if _, exists := s.items[name]; exists {
		return hysterr.AlreadyExistsFor(s.kind, name)
	}
	s.items[name] = item
	return nil
}

// Get returns a resource by name.
func (s *Store[T, PT]) Get(name string) (T, bool) {
	item, ok := s.items[name]
	return item, ok
}

// Update replaces an existing resource. Returns an error if not found.
func (s *Store[T, PT]) Update(name string, item T) error {
	if _, exists := s.items[name]; !exists {
		return hysterr.NotFoundFor(s.kind, name)
	}
	p := PT(&item)
	p.SetResourceName(name)
	s.items[name] = item
	return nil
}

// Delete removes a resource by name. Returns an error if not found.
func (s *Store[T, PT]) Delete(name string) error {
	if _, exists := s.items[name]; !exists {
		return hysterr.NotFoundFor(s.kind, name)
	}
	delete(s.items, name)
	return nil
}

// List returns all resources sorted by name (or custom sort if set).
func (s *Store[T, PT]) List() []T {
	items := make([]T, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	if s.sortFunc != nil {
		sort.Slice(items, func(i, j int) bool {
			return s.sortFunc(items[i], items[j])
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			return PT(&items[i]).ResourceName() < PT(&items[j]).ResourceName()
		})
	}
	return items
}

// Len returns the number of stored resources.
func (s *Store[T, PT]) Len() int {
	return len(s.items)
}

// Items returns the underlying map (for serialization).
func (s *Store[T, PT]) Items() map[string]T {
	return s.items
}

// SetItems replaces the underlying map (for deserialization).
// Populates each resource's Name from its map key.
func (s *Store[T, PT]) SetItems(items map[string]T) {
	if items == nil {
		s.items = make(map[string]T)
		return
	}
	s.items = items
	for name, item := range s.items {
		p := PT(&item)
		p.SetResourceName(name)
		s.items[name] = item
	}
}
