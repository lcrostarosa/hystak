package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hystak/hystak/internal/keyconfig"
	"github.com/spf13/cobra"
)

func newTestCmd(input string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

func TestPromptKeybindingProfile(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  keyconfig.Profile
	}{
		{"arrows explicit", "a\n", keyconfig.ProfileArrows},
		{"arrows default", "\n", keyconfig.ProfileArrows},
		{"vim", "v\n", keyconfig.ProfileVim},
		{"vim full", "vim\n", keyconfig.ProfileVim},
		{"classic", "c\n", keyconfig.ProfileClassic},
		{"classic full", "classic\n", keyconfig.ProfileClassic},
		{"unknown defaults to arrows", "x\n", keyconfig.ProfileArrows},
		{"uppercase", "V\n", keyconfig.ProfileVim},
		{"EOF defaults to arrows", "", keyconfig.ProfileArrows},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCmd(tt.input)
			got, err := promptKeybindingProfile(cmd)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("promptKeybindingProfile() = %q, want %q", got, tt.want)
			}
		})
	}
}
