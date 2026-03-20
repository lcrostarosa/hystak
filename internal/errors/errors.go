package hysterr

import (
	"errors"
	"fmt"
)

// NotFoundError indicates a named resource does not exist.
type NotFoundError struct {
	Kind string // "server", "project", "skill", "hook", "permission", "template", "tag"
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.Name)
}

// AlreadyExistsError indicates a named resource already exists.
type AlreadyExistsError struct {
	Kind string
	Name string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s %q already exists", e.Kind, e.Name)
}

// ReferencedError indicates a resource cannot be deleted because it is referenced.
type ReferencedError struct {
	Kind         string
	Name         string
	ReferencedBy string
}

func (e *ReferencedError) Error() string {
	return fmt.Sprintf("cannot delete %s %q: referenced by tag %q", e.Kind, e.Name, e.ReferencedBy)
}

// AlreadyAssignedError indicates a resource is already assigned to a project.
type AlreadyAssignedError struct {
	Kind        string
	Name        string
	ProjectName string
}

func (e *AlreadyAssignedError) Error() string {
	return fmt.Sprintf("%s %q already assigned to project %q", e.Kind, e.Name, e.ProjectName)
}

// NotAssignedError indicates a resource is not assigned to a project.
type NotAssignedError struct {
	Kind        string
	Name        string
	ProjectName string
}

func (e *NotAssignedError) Error() string {
	return fmt.Sprintf("%s %q not assigned to project %q", e.Kind, e.Name, e.ProjectName)
}

// --- Type checks ---

// IsNotFound reports whether err is a NotFoundError.
func IsNotFound(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

// IsAlreadyExists reports whether err is an AlreadyExistsError.
func IsAlreadyExists(err error) bool {
	var e *AlreadyExistsError
	return errors.As(err, &e)
}

// IsReferenced reports whether err is a ReferencedError.
func IsReferenced(err error) bool {
	var e *ReferencedError
	return errors.As(err, &e)
}

// IsAlreadyAssigned reports whether err is an AlreadyAssignedError.
func IsAlreadyAssigned(err error) bool {
	var e *AlreadyAssignedError
	return errors.As(err, &e)
}

// IsNotAssigned reports whether err is a NotAssignedError.
func IsNotAssigned(err error) bool {
	var e *NotAssignedError
	return errors.As(err, &e)
}

// --- Convenience constructors ---

func ServerNotFound(name string) error       { return &NotFoundError{Kind: "server", Name: name} }
func ProjectNotFound(name string) error      { return &NotFoundError{Kind: "project", Name: name} }
func ProfileNotFound(name string) error      { return &NotFoundError{Kind: "profile", Name: name} }
func SkillNotFound(name string) error        { return &NotFoundError{Kind: "skill", Name: name} }
func HookNotFound(name string) error         { return &NotFoundError{Kind: "hook", Name: name} }
func PermissionNotFound(name string) error   { return &NotFoundError{Kind: "permission", Name: name} }
func TemplateNotFound(name string) error     { return &NotFoundError{Kind: "template", Name: name} }
func TagNotFound(name string) error          { return &NotFoundError{Kind: "tag", Name: name} }

func ServerAlreadyExists(name string) error     { return &AlreadyExistsError{Kind: "server", Name: name} }
func ProjectAlreadyExists(name string) error    { return &AlreadyExistsError{Kind: "project", Name: name} }
func ProfileAlreadyExists(name string) error    { return &AlreadyExistsError{Kind: "profile", Name: name} }
func SkillAlreadyExists(name string) error      { return &AlreadyExistsError{Kind: "skill", Name: name} }
func HookAlreadyExists(name string) error       { return &AlreadyExistsError{Kind: "hook", Name: name} }
func PermissionAlreadyExists(name string) error { return &AlreadyExistsError{Kind: "permission", Name: name} }
func TemplateAlreadyExists(name string) error   { return &AlreadyExistsError{Kind: "template", Name: name} }
func TagAlreadyExists(name string) error        { return &AlreadyExistsError{Kind: "tag", Name: name} }

func ServerReferenced(name, tag string) error {
	return &ReferencedError{Kind: "server", Name: name, ReferencedBy: tag}
}

func ServerAlreadyAssigned(name, project string) error {
	return &AlreadyAssignedError{Kind: "server", Name: name, ProjectName: project}
}
func SkillAlreadyAssigned(name, project string) error {
	return &AlreadyAssignedError{Kind: "skill", Name: name, ProjectName: project}
}
func HookAlreadyAssigned(name, project string) error {
	return &AlreadyAssignedError{Kind: "hook", Name: name, ProjectName: project}
}
func PermissionAlreadyAssigned(name, project string) error {
	return &AlreadyAssignedError{Kind: "permission", Name: name, ProjectName: project}
}

func ServerNotAssigned(name, project string) error {
	return &NotAssignedError{Kind: "server", Name: name, ProjectName: project}
}
func SkillNotAssigned(name, project string) error {
	return &NotAssignedError{Kind: "skill", Name: name, ProjectName: project}
}
func HookNotAssigned(name, project string) error {
	return &NotAssignedError{Kind: "hook", Name: name, ProjectName: project}
}
func PermissionNotAssigned(name, project string) error {
	return &NotAssignedError{Kind: "permission", Name: name, ProjectName: project}
}
