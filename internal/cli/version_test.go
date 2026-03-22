package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd_DefaultValues(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := versionCmd
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "hystak dev") {
		t.Errorf("expected default version 'dev', got: %s", output)
	}
	if !strings.Contains(output, "commit: unknown") {
		t.Errorf("expected default commit 'unknown', got: %s", output)
	}
	if !strings.Contains(output, "built:  unknown") {
		t.Errorf("expected default date 'unknown', got: %s", output)
	}
}

func TestVersionCmd_CustomValues(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = origVersion, origCommit, origDate
	})

	version = "1.2.3"
	commit = "abc1234"
	date = "2026-03-22"

	buf := &bytes.Buffer{}
	cmd := versionCmd
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "hystak 1.2.3") {
		t.Errorf("expected version '1.2.3', got: %s", output)
	}
	if !strings.Contains(output, "commit: abc1234") {
		t.Errorf("expected commit 'abc1234', got: %s", output)
	}
	if !strings.Contains(output, "built:  2026-03-22") {
		t.Errorf("expected date '2026-03-22', got: %s", output)
	}
}
