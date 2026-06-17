package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAmbientDoctorHappyPath(t *testing.T) {
	projectDir := t.TempDir()
	withWorkingDir(t, projectDir)
	t.Setenv("TILLER_AMBIENT_DISABLED", "")

	out, err := captureStdout(func() error {
		return runAmbient([]string{"doctor"})
	})
	if err != nil {
		t.Fatalf("runAmbient doctor: %v\n%s", err, out)
	}
	for _, want := range []string{
		"PASS ambient runtime: executable ",
		" version " + Version,
		"PASS ambient bypass: not active",
		"PASS classifier smoke: ambient control status",
		"PASS classifier smoke: ambient control next",
		"PASS classifier smoke: ambient control step dry-run",
		"PASS classifier smoke: ambient control step without dry-run denied",
		"PASS classifier smoke: ambient control doctor",
		"PASS classifier smoke: ambient control doctor extra-arg denied",
		"PASS classifier smoke: git status readonly",
		"PASS classifier smoke: lsof port diagnostics readonly",
		"PASS classifier smoke: ss port diagnostics readonly",
		"PASS classifier smoke: go build denied-classified",
		"PASS fallback ledger smoke: write/read ok",
		`PASS hook smoke: PreToolUse Bash "git status --short" silent allow`,
		`PASS hook smoke: PreToolUse Bash "lsof -iTCP -sTCP:LISTEN -P -n" silent allow`,
		`PASS hook smoke: PreToolUse Bash "tiller ambient status" silent allow`,
		`PASS hook smoke: PreToolUse Bash "tiller ambient next" silent allow`,
		`PASS hook smoke: PreToolUse Bash "tiller ambient step --dry-run" silent allow`,
		`PASS hook smoke: PreToolUse Bash "tiller ambient doctor" silent allow`,
		`PASS hook smoke: PreToolUse Bash "tiller ambient step" denied without dry-run`,
		`PASS hook smoke: PreToolUse Bash "go build ./..." Codex deny guidance`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "FAIL ") {
		t.Fatalf("doctor output contains failure:\n%s", out)
	}
}

func TestRunAmbientDoctorBypassMarkerWarnsWithoutFailing(t *testing.T) {
	projectDir := t.TempDir()
	withWorkingDir(t, projectDir)
	t.Setenv("TILLER_AMBIENT_DISABLED", "")
	if err := os.MkdirAll(filepath.Join(projectDir, ".tiller"), 0o755); err != nil {
		t.Fatalf("mkdir .tiller: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".tiller", "ambient.disabled"), []byte("disabled\n"), 0o644); err != nil {
		t.Fatalf("write disabled marker: %v", err)
	}

	out, err := captureStdout(func() error {
		return runAmbient([]string{"doctor"})
	})
	if err != nil {
		t.Fatalf("runAmbient doctor should warn, not fail: %v\n%s", err, out)
	}
	if !strings.Contains(out, "WARN ambient bypass: enabled") {
		t.Fatalf("doctor output missing bypass warning:\n%s", out)
	}
	if strings.Contains(out, "FAIL ") {
		t.Fatalf("doctor output contains failure:\n%s", out)
	}
}

func TestRunAmbientDoctorPermissionModeBypassWarn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := t.TempDir()
	withWorkingDir(t, projectDir)
	t.Setenv("TILLER_AMBIENT_DISABLED", "")

	// Write a user-level settings.json with bypassPermissions.
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	settings := map[string]any{
		"permissions": map[string]any{
			"defaultMode": "bypassPermissions",
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	out, err := captureStdout(func() error {
		return runAmbient([]string{"doctor"})
	})
	if err != nil {
		t.Fatalf("runAmbient doctor should not fail for bypassPermissions WARN: %v\n%s", err, out)
	}
	if !strings.Contains(out, "WARN") || !strings.Contains(out, "bypassPermissions") {
		t.Fatalf("doctor output missing WARN for bypassPermissions:\n%s", out)
	}
	if !strings.Contains(out, "NOT reliably enforced") {
		t.Fatalf("doctor output missing not-reliably-enforced warning:\n%s", out)
	}
	if strings.Contains(out, "FAIL ") {
		t.Fatalf("doctor output contains failure (permission mode check must not fail):\n%s", out)
	}
}

func TestRunAmbientDoctorPermissionModeDefaultPass(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := t.TempDir()
	withWorkingDir(t, projectDir)
	t.Setenv("TILLER_AMBIENT_DISABLED", "")

	// Write a user-level settings.json with default mode.
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	settings := map[string]any{
		"permissions": map[string]any{
			"defaultMode": "default",
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	out, err := captureStdout(func() error {
		return runAmbient([]string{"doctor"})
	})
	if err != nil {
		t.Fatalf("runAmbient doctor should not fail for default mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "default") {
		t.Fatalf("doctor output missing PASS for default mode:\n%s", out)
	}
	if strings.Contains(out, "FAIL ") {
		t.Fatalf("doctor output contains failure:\n%s", out)
	}
}

func TestRunAmbientUsageErrorsMentionDoctor(t *testing.T) {
	if err := runAmbient(nil); err == nil || !strings.Contains(err.Error(), "usage: tiller ambient disable|enable|status|next|step --dry-run|doctor") {
		t.Fatalf("runAmbient(nil) err = %v", err)
	}
	if err := runAmbient([]string{"pause"}); err == nil || !strings.Contains(err.Error(), "disable, enable, status, next, step --dry-run, or doctor") {
		t.Fatalf("runAmbient(pause) err = %v", err)
	}
}
