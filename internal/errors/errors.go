package errors

import "fmt"

// ProjectNotFound indicates a project name was not found in the store.
type ProjectNotFound struct {
	Name string
}

func (e *ProjectNotFound) Error() string {
	return fmt.Sprintf("project %q not found", e.Name)
}

// ServerNotFound indicates a server name was not found in the registry.
type ServerNotFound struct {
	Name string
}

func (e *ServerNotFound) Error() string {
	return fmt.Sprintf("server %q not found", e.Name)
}

// ResourceNotFound indicates a named resource of a given kind was not found.
type ResourceNotFound struct {
	Kind string
	Name string
}

func (e *ResourceNotFound) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.Name)
}

// AlreadyExists indicates a resource with the given name already exists.
type AlreadyExists struct {
	Kind string
	Name string
}

func (e *AlreadyExists) Error() string {
	return fmt.Sprintf("%s %q already exists", e.Kind, e.Name)
}

// ValidationError indicates invalid input data.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ConfigParseError indicates a configuration file could not be parsed.
type ConfigParseError struct {
	Path string
	Err  error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse config %q: %v", e.Path, e.Err)
}

func (e *ConfigParseError) Unwrap() error {
	return e.Err
}
