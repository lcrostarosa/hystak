package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPermissionType_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value PermissionType
		want  bool
	}{
		{"allow", PermissionAllow, true},
		{"deny", PermissionDeny, true},
		{"empty", PermissionType(""), false},
		{"unknown", PermissionType("maybe"), false},
		{"case sensitive", PermissionType("Allow"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("PermissionType(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestPermissionRule_ResourceName(t *testing.T) {
	p := &PermissionRule{Name: "allow-bash"}
	if got := p.ResourceName(); got != "allow-bash" {
		t.Errorf("ResourceName() = %q, want %q", got, "allow-bash")
	}
	p.SetResourceName("deny-rm")
	if got := p.ResourceName(); got != "deny-rm" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "deny-rm")
	}
}

func TestPermissionRule_YAMLRoundTrip(t *testing.T) {
	original := PermissionRule{
		Name: "allow-bash",
		Rule: "Bash(*)",
		Type: PermissionAllow,
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored PermissionRule
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
