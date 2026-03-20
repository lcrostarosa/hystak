package model

import "testing"

func TestPermissionRule_EffectiveType_EmptyDefault(t *testing.T) {
	p := PermissionRule{Rule: "Bash(*)"}
	got := p.EffectiveType()
	if got != "allow" {
		t.Errorf("EffectiveType() = %q, want %q", got, "allow")
	}
}

func TestPermissionRule_EffectiveType_Allow(t *testing.T) {
	p := PermissionRule{Rule: "Bash(*)", Type: "allow"}
	got := p.EffectiveType()
	if got != "allow" {
		t.Errorf("EffectiveType() = %q, want %q", got, "allow")
	}
}

func TestPermissionRule_EffectiveType_Deny(t *testing.T) {
	p := PermissionRule{Rule: "WebFetch(domain:evil.com)", Type: "deny"}
	got := p.EffectiveType()
	if got != "deny" {
		t.Errorf("EffectiveType() = %q, want %q", got, "deny")
	}
}
