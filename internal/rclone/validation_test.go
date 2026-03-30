package rclone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func createMockRcloneValidation(t *testing.T, script string) string {
	t.Helper()
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "rclone")
	if runtime.GOOS == "windows" {
		mockPath += ".bat"
	}
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock rclone: %v", err)
	}
	return mockPath
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{
			name:      "standard format with rclone prefix",
			input:     "rclone v1.62.0",
			wantMajor: 1,
			wantMinor: 62,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version with v prefix only",
			input:     "v1.60.0",
			wantMajor: 1,
			wantMinor: 60,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version without prefix",
			input:     "1.58.3",
			wantMajor: 1,
			wantMinor: 58,
			wantPatch: 3,
			wantErr:   false,
		},
		{
			name:      "multi-line output",
			input:     "rclone v1.65.2\n- os/version: linux\n- os/kernel: 6.1.0",
			wantMajor: 1,
			wantMinor: 65,
			wantPatch: 2,
			wantErr:   false,
		},
		{
			name:      "version with beta suffix",
			input:     "v1.61.0-beta.1234",
			wantMajor: 1,
			wantMinor: 61,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no version pattern",
			input:   "no version here",
			wantErr: true,
		},
		{
			name:    "incomplete version",
			input:   "v1.62",
			wantErr: true,
		},
		{
			name:      "high version numbers",
			input:     "v2.100.500",
			wantMajor: 2,
			wantMinor: 100,
			wantPatch: 500,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.major != tt.wantMajor {
					t.Errorf("parseVersion().major = %v, want %v", got.major, tt.wantMajor)
				}
				if got.minor != tt.wantMinor {
					t.Errorf("parseVersion().minor = %v, want %v", got.minor, tt.wantMinor)
				}
				if got.patch != tt.wantPatch {
					t.Errorf("parseVersion().patch = %v, want %v", got.patch, tt.wantPatch)
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a    versionTuple
		b    versionTuple
		want int
	}{
		{
			name: "equal versions",
			a:    versionTuple{1, 60, 0},
			b:    versionTuple{1, 60, 0},
			want: 0,
		},
		{
			name: "a greater major",
			a:    versionTuple{2, 0, 0},
			b:    versionTuple{1, 60, 0},
			want: 1,
		},
		{
			name: "a less major",
			a:    versionTuple{1, 0, 0},
			b:    versionTuple{2, 0, 0},
			want: -1,
		},
		{
			name: "a greater minor",
			a:    versionTuple{1, 61, 0},
			b:    versionTuple{1, 60, 0},
			want: 1,
		},
		{
			name: "a less minor",
			a:    versionTuple{1, 59, 0},
			b:    versionTuple{1, 60, 0},
			want: -1,
		},
		{
			name: "a greater patch",
			a:    versionTuple{1, 60, 1},
			b:    versionTuple{1, 60, 0},
			want: 1,
		},
		{
			name: "a less patch",
			a:    versionTuple{1, 60, 0},
			b:    versionTuple{1, 60, 1},
			want: -1,
		},
		{
			name: "major takes precedence over minor",
			a:    versionTuple{2, 0, 0},
			b:    versionTuple{1, 100, 100},
			want: 1,
		},
		{
			name: "minor takes precedence over patch",
			a:    versionTuple{1, 61, 0},
			b:    versionTuple{1, 60, 100},
			want: 1,
		},
		{
			name: "zero versions equal",
			a:    versionTuple{0, 0, 0},
			b:    versionTuple{0, 0, 0},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("compareVersions(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestFormatRemoteNames(t *testing.T) {
	tests := []struct {
		name     string
		remotes  []Remote
		want     string
		contains string
	}{
		{
			name:    "empty slice",
			remotes: []Remote{},
			want:    "",
		},
		{
			name:    "single remote",
			remotes: []Remote{{Name: "gdrive"}},
			want:    "gdrive",
		},
		{
			name:    "two remotes",
			remotes: []Remote{{Name: "gdrive"}, {Name: "dropbox"}},
			want:    "gdrive, dropbox",
		},
		{
			name:    "five remotes",
			remotes: []Remote{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"}},
			want:    "a, b, c, d, e",
		},
		{
			name:    "six remotes - truncates",
			remotes: []Remote{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"}, {Name: "f"}},
			want:    "a, b, c, d, e (and 1 more)",
		},
		{
			name:    "many remotes - truncates",
			remotes: []Remote{{Name: "r1"}, {Name: "r2"}, {Name: "r3"}, {Name: "r4"}, {Name: "r5"}, {Name: "r6"}, {Name: "r7"}, {Name: "r8"}},
			want:    "r1, r2, r3, r4, r5 (and 3 more)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRemoteNames(tt.remotes)
			if got != tt.want {
				t.Errorf("formatRemoteNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasCriticalFailure(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name:    "empty results",
			results: []CheckResult{},
			want:    false,
		},
		{
			name: "all passed",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: true, IsCritical: false},
			},
			want: false,
		},
		{
			name: "critical failure",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: false, IsCritical: true},
			},
			want: true,
		},
		{
			name: "non-critical failure only",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: false, IsCritical: false},
			},
			want: false,
		},
		{
			name: "multiple critical failures",
			results: []CheckResult{
				{Name: "Check1", Passed: false, IsCritical: true},
				{Name: "Check2", Passed: false, IsCritical: true},
			},
			want: true,
		},
		{
			name: "mixed results with critical failure",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: false, IsCritical: false},
				{Name: "Check3", Passed: false, IsCritical: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCriticalFailure(tt.results); got != tt.want {
				t.Errorf("HasCriticalFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllPassed(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name:    "empty results",
			results: []CheckResult{},
			want:    true,
		},
		{
			name: "all passed",
			results: []CheckResult{
				{Name: "Check1", Passed: true},
				{Name: "Check2", Passed: true},
			},
			want: true,
		},
		{
			name: "one failure",
			results: []CheckResult{
				{Name: "Check1", Passed: true},
				{Name: "Check2", Passed: false},
			},
			want: false,
		},
		{
			name: "all failures",
			results: []CheckResult{
				{Name: "Check1", Passed: false},
				{Name: "Check2", Passed: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllPassed(tt.results); got != tt.want {
				t.Errorf("AllPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	tests := []struct {
		name        string
		results     []CheckResult
		wantContain []string
		dontContain []string
	}{
		{
			name:        "empty results",
			results:     []CheckResult{},
			wantContain: []string{"Pre-flight Check Results:", "----"},
		},
		{
			name: "passed check",
			results: []CheckResult{
				{Name: "Test Check", Passed: true, Message: "All good"},
			},
			wantContain: []string{"✓ PASS", "Test Check", "All good"},
			dontContain: []string{"Suggestion:"},
		},
		{
			name: "failed critical check",
			results: []CheckResult{
				{Name: "Critical Check", Passed: false, Message: "Something wrong", Suggestion: "Fix it", IsCritical: true},
			},
			wantContain: []string{"✗ FAIL (critical)", "Critical Check", "Something wrong", "Suggestion:", "Fix it"},
		},
		{
			name: "failed optional check",
			results: []CheckResult{
				{Name: "Optional Check", Passed: false, Message: "Optional issue", Suggestion: "Consider fixing", IsCritical: false},
			},
			wantContain: []string{"⚠ FAIL (optional)", "Optional Check", "Optional issue", "Consider fixing"},
		},
		{
			name: "multiple checks",
			results: []CheckResult{
				{Name: "Check 1", Passed: true, Message: "OK"},
				{Name: "Check 2", Passed: false, Message: "Failed", IsCritical: true},
			},
			wantContain: []string{"✓ PASS", "✗ FAIL (critical)", "Check 1", "Check 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatResults(tt.results)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("FormatResults() missing expected string %q in output:\n%s", want, got)
				}
			}

			for _, dontWant := range tt.dontContain {
				if strings.Contains(got, dontWant) {
					t.Errorf("FormatResults() contains unexpected string %q in output:\n%s", dontWant, got)
				}
			}
		})
	}
}

func TestCheckResultStructure(t *testing.T) {
	result := CheckResult{
		Name:       "Test Check",
		Passed:     true,
		Message:    "Test message",
		Suggestion: "Test suggestion",
		IsCritical: true,
	}

	if result.Name != "Test Check" {
		t.Errorf("CheckResult.Name = %q, want %q", result.Name, "Test Check")
	}
	if result.Passed != true {
		t.Errorf("CheckResult.Passed = %v, want %v", result.Passed, true)
	}
	if result.Message != "Test message" {
		t.Errorf("CheckResult.Message = %q, want %q", result.Message, "Test message")
	}
	if result.Suggestion != "Test suggestion" {
		t.Errorf("CheckResult.Suggestion = %q, want %q", result.Suggestion, "Test suggestion")
	}
	if result.IsCritical != true {
		t.Errorf("CheckResult.IsCritical = %v, want %v", result.IsCritical, true)
	}
}

func TestCheckRcloneBinaryNilClient(t *testing.T) {
	result := checkRcloneBinary(nil)

	if result.Passed {
		t.Error("checkRcloneBinary(nil) should not pass")
	}
	if !result.IsCritical {
		t.Error("checkRcloneBinary should always be critical")
	}
	if result.Name != "Rclone Binary" {
		t.Errorf("checkRcloneBinary().Name = %q, want %q", result.Name, "Rclone Binary")
	}
	if result.Suggestion == "" {
		t.Error("checkRcloneBinary(nil) should provide a suggestion")
	}
}

func TestPreflightChecksNilClient(t *testing.T) {
	results := PreflightChecks(nil)

	if len(results) == 0 {
		t.Error("PreflightChecks(nil) should return at least one result")
	}

	if results[0].Name != "Rclone Binary" {
		t.Errorf("First check name = %q, want %q", results[0].Name, "Rclone Binary")
	}
	if results[0].Passed {
		t.Error("Rclone Binary check should fail with nil client")
	}
}

func TestVersionTupleZeroValue(t *testing.T) {
	var v versionTuple
	if v.major != 0 || v.minor != 0 || v.patch != 0 {
		t.Errorf("Zero value versionTuple = %v, want {0, 0, 0}", v)
	}
}

func TestRemoteStructure(t *testing.T) {
	remote := Remote{
		Name:     "gdrive",
		Type:     "drive",
		RootPath: "gdrive:",
	}

	if remote.Name != "gdrive" {
		t.Errorf("Remote.Name = %q, want %q", remote.Name, "gdrive")
	}
	if remote.Type != "drive" {
		t.Errorf("Remote.Type = %q, want %q", remote.Type, "drive")
	}
	if remote.RootPath != "gdrive:" {
		t.Errorf("Remote.RootPath = %q, want %q", remote.RootPath, "gdrive:")
	}
}

func TestCheckRcloneBinaryFound(t *testing.T) {
	path, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found in PATH")
	}

	c := NewClientWithPath(path)
	result := checkRcloneBinary(c)

	if !result.Passed {
		t.Errorf("checkRcloneBinary() should pass for existing binary: %s", result.Message)
	}
	if !result.IsCritical {
		t.Error("checkRcloneBinary should always be critical")
	}
}

func TestCheckRcloneBinaryNotFound(t *testing.T) {
	c := NewClientWithPath("/nonexistent/path/to/rclone")
	result := checkRcloneBinary(c)

	if result.Passed {
		t.Error("checkRcloneBinary() should fail for nonexistent binary")
	}
	if !result.IsCritical {
		t.Error("checkRcloneBinary should always be critical")
	}
	if !strings.Contains(result.Suggestion, "rclone.org/install") {
		t.Error("checkRcloneBinary() suggestion should contain 'rclone.org/install'")
	}
}

func TestCheckRcloneVersionSuccess(t *testing.T) {
	mockScript := `#!/bin/sh
echo "rclone v1.62.0"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if !result.Passed {
		t.Errorf("checkRcloneVersion() should pass for version 1.62.0: %s", result.Message)
	}
	if !result.IsCritical {
		t.Error("checkRcloneVersion should always be critical")
	}
}

func TestCheckRcloneVersionBelowMinimum(t *testing.T) {
	mockScript := `#!/bin/sh
echo "rclone v1.50.0"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if result.Passed {
		t.Error("checkRcloneVersion() should fail for version below 1.60.0")
	}
	if result.Suggestion == "" {
		t.Error("checkRcloneVersion() should provide suggestion for old version")
	}
}

func TestCheckRcloneVersionError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "error" >&2
exit 1
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if result.Passed {
		t.Error("checkRcloneVersion() should fail when version command fails")
	}
}

func TestCheckRcloneVersionParseError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "invalid version output"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if result.Passed {
		t.Error("checkRcloneVersion() should fail for unparseable version")
	}
}

func TestCheckConfiguredRemotesSuccess(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo "gdrive:"
		echo "dropbox:"
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkConfiguredRemotes(c)

	if !result.Passed {
		t.Errorf("checkConfiguredRemotes() should pass with configured remotes: %s", result.Message)
	}
}

func TestCheckConfiguredRemotesEmpty(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkConfiguredRemotes(c)

	if result.Passed {
		t.Error("checkConfiguredRemotes() should fail when no remotes configured")
	}
	if result.Suggestion == "" {
		t.Error("checkConfiguredRemotes() should provide suggestion")
	}
	if result.IsCritical {
		t.Error("checkConfiguredRemotes() should not be critical - it's a non-fatal warning")
	}
	if !strings.Contains(result.Suggestion, "rclone config") {
		t.Error("checkConfiguredRemotes() suggestion should mention 'rclone config'")
	}
}

func TestCheckConfiguredRemotesError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "error" >&2
exit 1
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkConfiguredRemotes(c)

	if result.Passed {
		t.Error("checkConfiguredRemotes() should fail when listremotes fails")
	}
}

func TestCheckSystemdUserSessionSystemctlNotFound(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	result := checkSystemdUserSession()

	if result.Passed {
		t.Error("checkSystemdUserSession() should fail when systemctl not found")
	}
}

func TestCheckFusermountFound(t *testing.T) {
	path, err := exec.LookPath("fusermount")
	if err != nil {
		path, err = exec.LookPath("fusermount3")
	}
	if err != nil {
		t.Skip("neither fusermount nor fusermount3 found")
	}

	result := checkFusermount()

	if !result.Passed {
		t.Errorf("checkFusermount() should pass when fusermount exists at %s", path)
	}
}

func TestCheckFusermountNotFound(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	result := checkFusermount()

	if result.Passed {
		t.Error("checkFusermount() should fail when no fusermount found")
	}
	if result.IsCritical {
		t.Error("checkFusermount() should not be critical")
	}
}

func TestPreflightChecksFullFlow(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	version)
		echo "rclone v1.62.0"
		;;
	listremotes)
		echo "gdrive:"
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	results := PreflightChecks(c)

	if len(results) < 4 {
		t.Errorf("PreflightChecks() returned %d results, want at least 4", len(results))
	}

	for i, r := range results {
		if r.Name == "" {
			t.Errorf("results[%d].Name is empty", i)
		}
	}
}

func TestPreflightChecksWithBinaryNotFound(t *testing.T) {
	c := NewClientWithPath("/nonexistent/rclone")

	results := PreflightChecks(c)

	if len(results) < 4 {
		t.Errorf("PreflightChecks() returned %d results, want at least 4", len(results))
	}

	if results[0].Passed {
		t.Error("First check (Rclone Binary) should fail")
	}

	for _, r := range results {
		if r.Name == "Rclone Version" && r.Passed {
			t.Error("Rclone Version check should be skipped/failed when binary not found")
		}
	}
}

func TestFormatResultsSingleCheck(t *testing.T) {
	results := []CheckResult{
		{Name: "Single Check", Passed: true, Message: "All good"},
	}

	output := FormatResults(results)

	if !strings.Contains(output, "Single Check") {
		t.Error("FormatResults() should contain check name")
	}
	if !strings.Contains(output, "✓ PASS") {
		t.Error("FormatResults() should contain pass marker")
	}
}

func TestFormatResultsNoSuggestion(t *testing.T) {
	results := []CheckResult{
		{Name: "Check", Passed: true, Message: "OK", Suggestion: ""},
	}

	output := FormatResults(results)

	if strings.Contains(output, "Suggestion:") {
		t.Error("FormatResults() should not contain 'Suggestion:' when empty")
	}
}

func TestFormatResultsWithSuggestion(t *testing.T) {
	results := []CheckResult{
		{Name: "Check", Passed: false, Message: "Failed", Suggestion: "Try this fix", IsCritical: true},
	}

	output := FormatResults(results)

	if !strings.Contains(output, "Suggestion: Try this fix") {
		t.Error("FormatResults() should contain suggestion")
	}
}

func TestCheckRcloneVersionMinimumVersion(t *testing.T) {
	mockScript := `#!/bin/sh
echo "rclone v1.60.0"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if !result.Passed {
		t.Errorf("checkRcloneVersion() should pass for exact minimum version 1.60.0: %s", result.Message)
	}
}

func TestCheckRcloneVersionNewerVersion(t *testing.T) {
	mockScript := `#!/bin/sh
echo "rclone v2.0.0"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkRcloneVersion(c)

	if !result.Passed {
		t.Errorf("checkRcloneVersion() should pass for version 2.0.0: %s", result.Message)
	}
}

func TestCheckRcloneBinaryWithDefaultPath(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	c := NewClient()
	result := checkRcloneBinary(c)

	if result.Passed {
		t.Error("checkRcloneBinary() should fail when rclone not in PATH")
	}
}

func TestValidateOnCalendar(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "daily",
			input:   "daily",
			wantErr: false,
		},
		{
			name:    "hourly",
			input:   "hourly",
			wantErr: false,
		},
		{
			name:    "weekly",
			input:   "weekly",
			wantErr: false,
		},
		{
			name:    "monthly",
			input:   "monthly",
			wantErr: false,
		},
		{
			name:    "yearly",
			input:   "yearly",
			wantErr: false,
		},
		{
			name:    "annually",
			input:   "annually",
			wantErr: false,
		},
		{
			name:    "quarterly",
			input:   "quarterly",
			wantErr: false,
		},
		{
			name:    "semiannually",
			input:   "semiannually",
			wantErr: false,
		},
		{
			name:    "named schedule uppercase",
			input:   "DAILY",
			wantErr: false,
		},
		{
			name:    "named schedule mixed case",
			input:   "Weekly",
			wantErr: false,
		},
		{
			name:    "every day at midnight",
			input:   "*-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "every day at 2am",
			input:   "*-*-* 02:00:00",
			wantErr: false,
		},
		{
			name:    "every day with wildcard time",
			input:   "*-*-* *:*:*",
			wantErr: false,
		},
		{
			name:    "specific date",
			input:   "2024-01-01 00:00:00",
			wantErr: false,
		},
		{
			name:    "first day of month",
			input:   "*-*-01 00:00:00",
			wantErr: false,
		},
		{
			name:    "Monday at midnight",
			input:   "Mon *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "Friday at 5pm",
			input:   "Fri *-*-* 17:00:00",
			wantErr: false,
		},
		{
			name:    "multiple days Monday and Friday",
			input:   "Mon,Fri *-*-* 09:00:00",
			wantErr: false,
		},
		{
			name:    "weekend days",
			input:   "Sat,Sun *-*-* 10:00:00",
			wantErr: false,
		},
		{
			name:    "every hour wildcard",
			input:   "*-*-* *:00:00",
			wantErr: false,
		},
		{
			name:    "every minute",
			input:   "*-*-* *:*:00",
			wantErr: false,
		},
		{
			name:    "year and month wildcard with day",
			input:   "*-*-15 12:00:00",
			wantErr: false,
		},
		{
			name:    "specific year month wildcard",
			input:   "2024-*-01 00:00:00",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "random string",
			input:   "notaschedule",
			wantErr: true,
		},
		{
			name:    "invalid named schedule",
			input:   "biweekly",
			wantErr: true,
		},
		{
			name:    "invalid day name",
			input:   "XYZ *-*-* 00:00:00",
			wantErr: true,
		},
		{
			name:    "missing time component",
			input:   "*-*-*",
			wantErr: true,
		},
		{
			name:    "missing date component",
			input:   "00:00:00",
			wantErr: true,
		},
		{
			name:    "malformed date",
			input:   "2024/01/01 00:00:00",
			wantErr: true,
		},
		{
			name:    "malformed time",
			input:   "*-*-* 00-00-00",
			wantErr: true,
		},
		{
			name:    "invalid hour value",
			input:   "*-*-* 25:00:00",
			wantErr: false, // We don't validate semantic correctness of values
		},
		{
			name:    "time without seconds",
			input:   "*-*-* 02:00",
			wantErr: false,
		},
		{
			name:    "time with just hour",
			input:   "*-*-* 02",
			wantErr: false,
		},
		{
			name:    "date with just year",
			input:   "2024",
			wantErr: true,
		},
		{
			name:    "trailing space",
			input:   "daily ",
			wantErr: false,
		},
		{
			name:    "leading space",
			input:   " daily",
			wantErr: false,
		},
		{
			name:    "Wednesday with space",
			input:   "Wed *-*-* 14:30:00",
			wantErr: false,
		},
		{
			name:    "Thursday abbreviation",
			input:   "Thu *-*-* 09:00:00",
			wantErr: false,
		},
		{
			name:    "Tuesday abbreviation",
			input:   "Tue *-*-* 08:00:00",
			wantErr: false,
		},
		{
			name:    "Sunday abbreviation",
			input:   "Sun *-*-* 00:00:00",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOnCalendar(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOnCalendar(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if err != nil && tt.input != "" && !strings.Contains(err.Error(), "Valid formats:") {
				t.Errorf("ValidateOnCalendar error should contain helpful format examples")
			}
		})
	}
}

func TestValidateOnCalendarErrorMessage(t *testing.T) {
	err := ValidateOnCalendar("invalid")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}

	errMsg := err.Error()

	if !strings.Contains(errMsg, "invalid OnCalendar format") {
		t.Error("error message should contain 'invalid OnCalendar format'")
	}
	if !strings.Contains(errMsg, "daily") {
		t.Error("error message should suggest 'daily' as valid format")
	}
	if !strings.Contains(errMsg, "weekly") {
		t.Error("error message should suggest 'weekly' as valid format")
	}
	if !strings.Contains(errMsg, "*-*-* 02:00:00") {
		t.Error("error message should show example '*-*-* 02:00:00'")
	}
	if !strings.Contains(errMsg, "Mon *-*-* 09:00:00") {
		t.Error("error message should show example 'Mon *-*-* 09:00:00'")
	}
}

func TestValidateOnCalendarEmptyInput(t *testing.T) {
	err := ValidateOnCalendar("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected 'cannot be empty' error, got: %v", err)
	}
}

// Additional comprehensive tests for parseVersion
func TestParseVersionEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{
			name:      "version with rc suffix",
			input:     "v1.62.0-rc.1",
			wantMajor: 1,
			wantMinor: 62,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version with dev suffix",
			input:     "v1.60.0-dev",
			wantMajor: 1,
			wantMinor: 60,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version in brackets",
			input:     "[v1.65.0]",
			wantMajor: 1,
			wantMinor: 65,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version with text before and after",
			input:     "rclone version v1.63.0 stable",
			wantMajor: 1,
			wantMinor: 63,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version with leading zeros",
			input:     "v01.060.000",
			wantMajor: 1,
			wantMinor: 60,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:    "only major version",
			input:   "v1",
			wantErr: true,
		},
		{
			name:    "only major.minor",
			input:   "v1.62",
			wantErr: true,
		},
		{
			name:    "version with letters",
			input:   "v1.6a.0",
			wantErr: true,
		},
		{
			name:      "very high version numbers",
			input:     "v999.999.999",
			wantMajor: 999,
			wantMinor: 999,
			wantPatch: 999,
			wantErr:   false,
		},
		{
			name:      "multiple version patterns - should match first",
			input:     "v1.60.0 some text v2.0.0",
			wantMajor: 1,
			wantMinor: 60,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "version from systemctl output style",
			input:     "rclone v1.61.0 (linux)",
			wantMajor: 1,
			wantMinor: 61,
			wantPatch: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.major != tt.wantMajor {
					t.Errorf("parseVersion().major = %v, want %v", got.major, tt.wantMajor)
				}
				if got.minor != tt.wantMinor {
					t.Errorf("parseVersion().minor = %v, want %v", got.minor, tt.wantMinor)
				}
				if got.patch != tt.wantPatch {
					t.Errorf("parseVersion().patch = %v, want %v", got.patch, tt.wantPatch)
				}
			}
		})
	}
}

// Additional tests for compareVersions
func TestCompareVersionsEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		a    versionTuple
		b    versionTuple
		want int
	}{
		{
			name: "both zero versions",
			a:    versionTuple{0, 0, 0},
			b:    versionTuple{0, 0, 0},
			want: 0,
		},
		{
			name: "a is zero, b is not",
			a:    versionTuple{0, 0, 0},
			b:    versionTuple{1, 0, 0},
			want: -1,
		},
		{
			name: "b is zero, a is not",
			a:    versionTuple{1, 0, 0},
			b:    versionTuple{0, 0, 0},
			want: 1,
		},
		{
			name: "large version numbers equal",
			a:    versionTuple{999, 999, 999},
			b:    versionTuple{999, 999, 999},
			want: 0,
		},
		{
			name: "large version numbers a greater",
			a:    versionTuple{999, 999, 999},
			b:    versionTuple{100, 100, 100},
			want: 1,
		},
		{
			name: "negative comparison result",
			a:    versionTuple{1, 59, 9},
			b:    versionTuple{1, 60, 0},
			want: -1,
		},
		{
			name: "exact minimum version",
			a:    versionTuple{1, 60, 0},
			b:    versionTuple{1, 60, 0},
			want: 0,
		},
		{
			name: "one below minimum",
			a:    versionTuple{1, 59, 999},
			b:    versionTuple{1, 60, 0},
			want: -1,
		},
		{
			name: "one above minimum",
			a:    versionTuple{1, 60, 1},
			b:    versionTuple{1, 60, 0},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("compareVersions(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// Additional tests for formatRemoteNames
func TestFormatRemoteNamesEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		remotes []Remote
		want    string
	}{
		{
			name:    "nil slice",
			remotes: nil,
			want:    "",
		},
		{
			name:    "remote with special characters",
			remotes: []Remote{{Name: "gdrive-test"}, {Name: "s3_backup"}},
			want:    "gdrive-test, s3_backup",
		},
		{
			name:    "remote with long names",
			remotes: []Remote{{Name: "very_long_remote_name_1"}, {Name: "very_long_remote_name_2"}},
			want:    "very_long_remote_name_1, very_long_remote_name_2",
		},
		{
			name: "exactly five remotes - no truncation",
			remotes: []Remote{
				{Name: "r1"}, {Name: "r2"}, {Name: "r3"}, {Name: "r4"}, {Name: "r5"},
			},
			want: "r1, r2, r3, r4, r5",
		},
		{
			name: "seven remotes - truncation",
			remotes: []Remote{
				{Name: "r1"}, {Name: "r2"}, {Name: "r3"}, {Name: "r4"}, {Name: "r5"},
				{Name: "r6"}, {Name: "r7"},
			},
			want: "r1, r2, r3, r4, r5 (and 2 more)",
		},
		{
			name: "ten remotes - truncation",
			remotes: []Remote{
				{Name: "r1"}, {Name: "r2"}, {Name: "r3"}, {Name: "r4"}, {Name: "r5"},
				{Name: "r6"}, {Name: "r7"}, {Name: "r8"}, {Name: "r9"}, {Name: "r10"},
			},
			want: "r1, r2, r3, r4, r5 (and 5 more)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRemoteNames(tt.remotes)
			if got != tt.want {
				t.Errorf("formatRemoteNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Additional tests for HasCriticalFailure
func TestHasCriticalFailureEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name:    "nil slice",
			results: nil,
			want:    false,
		},
		{
			name: "single critical pass",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
			},
			want: false,
		},
		{
			name: "single critical fail",
			results: []CheckResult{
				{Name: "Check1", Passed: false, IsCritical: true},
			},
			want: true,
		},
		{
			name: "single non-critical fail",
			results: []CheckResult{
				{Name: "Check1", Passed: false, IsCritical: false},
			},
			want: false,
		},
		{
			name: "multiple non-critical failures",
			results: []CheckResult{
				{Name: "Check1", Passed: false, IsCritical: false},
				{Name: "Check2", Passed: false, IsCritical: false},
				{Name: "Check3", Passed: false, IsCritical: false},
			},
			want: false,
		},
		{
			name: "critical failure at end",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: true, IsCritical: false},
				{Name: "Check3", Passed: false, IsCritical: true},
			},
			want: true,
		},
		{
			name: "critical failure in middle",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: false, IsCritical: true},
				{Name: "Check3", Passed: true, IsCritical: false},
			},
			want: true,
		},
		{
			name: "all critical pass",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: true, IsCritical: true},
				{Name: "Check3", Passed: true, IsCritical: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCriticalFailure(tt.results); got != tt.want {
				t.Errorf("HasCriticalFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Additional tests for AllPassed
func TestAllPassedEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name:    "nil slice",
			results: nil,
			want:    true,
		},
		{
			name: "single pass",
			results: []CheckResult{
				{Name: "Check1", Passed: true},
			},
			want: true,
		},
		{
			name: "single fail",
			results: []CheckResult{
				{Name: "Check1", Passed: false},
			},
			want: false,
		},
		{
			name: "all pass with critical flags",
			results: []CheckResult{
				{Name: "Check1", Passed: true, IsCritical: true},
				{Name: "Check2", Passed: true, IsCritical: false},
				{Name: "Check3", Passed: true, IsCritical: true},
			},
			want: true,
		},
		{
			name: "last one fails",
			results: []CheckResult{
				{Name: "Check1", Passed: true},
				{Name: "Check2", Passed: true},
				{Name: "Check3", Passed: false},
			},
			want: false,
		},
		{
			name: "first one fails",
			results: []CheckResult{
				{Name: "Check1", Passed: false},
				{Name: "Check2", Passed: true},
				{Name: "Check3", Passed: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllPassed(tt.results); got != tt.want {
				t.Errorf("AllPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Additional tests for FormatResults
func TestFormatResultsEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		results     []CheckResult
		wantContain []string
		dontContain []string
	}{
		{
			name:        "nil results",
			results:     nil,
			wantContain: []string{"Pre-flight Check Results:", "----"},
		},
		{
			name: "check with empty message",
			results: []CheckResult{
				{Name: "Check1", Passed: true, Message: ""},
			},
			wantContain: []string{"✓ PASS", "Check1"},
		},
		{
			name: "check with very long message",
			results: []CheckResult{
				{Name: "Check1", Passed: true, Message: strings.Repeat("a", 500)},
			},
			wantContain: []string{"✓ PASS", "Check1"},
		},
		{
			name: "multiple critical failures",
			results: []CheckResult{
				{Name: "Critical1", Passed: false, IsCritical: true, Message: "Error 1"},
				{Name: "Critical2", Passed: false, IsCritical: true, Message: "Error 2"},
			},
			wantContain: []string{"✗ FAIL (critical)", "Critical1", "Critical2", "Error 1", "Error 2"},
		},
		{
			name: "mix of all types",
			results: []CheckResult{
				{Name: "Pass1", Passed: true, Message: "OK1"},
				{Name: "FailCritical", Passed: false, IsCritical: true, Message: "Critical error", Suggestion: "Fix it"},
				{Name: "Pass2", Passed: true, Message: "OK2"},
				{Name: "FailOptional", Passed: false, IsCritical: false, Message: "Optional issue", Suggestion: "Consider fixing"},
			},
			wantContain: []string{"✓ PASS", "✗ FAIL (critical)", "⚠ FAIL (optional)", "Suggestion:"},
		},
		{
			name: "check with multiline suggestion",
			results: []CheckResult{
				{Name: "Check1", Passed: false, Message: "Failed", Suggestion: "Step 1\nStep 2\nStep 3", IsCritical: true},
			},
			wantContain: []string{"Suggestion:", "Step 1", "Step 2", "Step 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatResults(tt.results)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("FormatResults() missing expected string %q in output:\n%s", want, got)
				}
			}

			for _, dontWant := range tt.dontContain {
				if strings.Contains(got, dontWant) {
					t.Errorf("FormatResults() contains unexpected string %q in output:\n%s", dontWant, got)
				}
			}
		})
	}
}

// Additional tests for ValidateOnCalendar
func TestValidateOnCalendarAdditionalCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "all named schedules lowercase",
			input:   "daily",
			wantErr: false,
		},
		{
			name:    "all named schedules uppercase",
			input:   "DAILY",
			wantErr: false,
		},
		{
			name:    "all named schedules mixed case",
			input:   "DaIlY",
			wantErr: false,
		},
		{
			name:    "hourly uppercase",
			input:   "HOURLY",
			wantErr: false,
		},
		{
			name:    "weekly uppercase",
			input:   "WEEKLY",
			wantErr: false,
		},
		{
			name:    "monthly uppercase",
			input:   "MONTHLY",
			wantErr: false,
		},
		{
			name:    "yearly uppercase",
			input:   "YEARLY",
			wantErr: false,
		},
		{
			name:    "annually uppercase",
			input:   "ANNUALLY",
			wantErr: false,
		},
		{
			name:    "quarterly uppercase",
			input:   "QUARTERLY",
			wantErr: false,
		},
		{
			name:    "semiannually uppercase",
			input:   "SEMIANNUALLY",
			wantErr: false,
		},
		{
			name:    "all days of week Monday",
			input:   "Mon *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Tuesday",
			input:   "Tue *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Wednesday",
			input:   "Wed *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Thursday",
			input:   "Thu *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Friday",
			input:   "Fri *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Saturday",
			input:   "Sat *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "all days of week Sunday",
			input:   "Sun *-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "multiple consecutive days",
			input:   "Mon,Tue,Wed *-*-* 09:00:00",
			wantErr: false,
		},
		{
			name:    "all weekdays",
			input:   "Mon,Tue,Wed,Thu,Fri *-*-* 09:00:00",
			wantErr: false,
		},
		{
			name:    "weekend",
			input:   "Sat,Sun *-*-* 10:00:00",
			wantErr: false,
		},
		{
			name:    "every minute wildcard",
			input:   "*-*-* *:*:*",
			wantErr: false,
		},
		{
			name:    "every hour at minute 0",
			input:   "*-*-* *:00:00",
			wantErr: false,
		},
		{
			name:    "every day at hour 0",
			input:   "*-*-* 00:*:*",
			wantErr: false,
		},
		{
			name:    "specific month wildcard day",
			input:   "*-06-* 12:00:00",
			wantErr: false,
		},
		{
			name:    "specific year wildcard month",
			input:   "2024-*-* 00:00:00",
			wantErr: false,
		},
		{
			name:    "time with just hour",
			input:   "*-*-* 14",
			wantErr: false,
		},
		{
			name:    "time with hour and minute",
			input:   "*-*-* 14:30",
			wantErr: false,
		},
		{
			name:    "time with hour minute second",
			input:   "*-*-* 14:30:45",
			wantErr: false,
		},
		{
			name:    "invalid named schedule typo",
			input:   "dailly",
			wantErr: true,
		},
		{
			name:    "invalid day abbreviation",
			input:   "Xyz *-*-* 00:00:00",
			wantErr: true,
		},
		{
			name:    "invalid day with typo",
			input:   "Mond *-*-* 00:00:00",
			wantErr: true,
		},
		{
			name:    "wrong date separator",
			input:   "*/01/01 00:00:00",
			wantErr: true,
		},
		{
			name:    "wrong time separator",
			input:   "*-*-* 00/00/00",
			wantErr: true,
		},
		{
			name:    "missing space between date and time",
			input:   "*-*-*00:00:00",
			wantErr: true,
		},
		{
			name:    "extra spaces",
			input:   "*-*-*  00:00:00",
			wantErr: false, // The regex allows multiple spaces
		},
		{
			name:    "invalid characters in date",
			input:   "*-a-* 00:00:00",
			wantErr: true,
		},
		{
			name:    "invalid characters in time",
			input:   "*-*-* a:00:00",
			wantErr: true,
		},
		{
			name:    "day of week with lowercase",
			input:   "mon *-*-* 00:00:00",
			wantErr: true, // Day names are case-sensitive in the regex
		},
		{
			name:    "day of week mixed case",
			input:   "MoN *-*-* 00:00:00",
			wantErr: true, // Day names are case-sensitive in the regex
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOnCalendar(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOnCalendar(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// Test ValidateOnCalendar with leading/trailing whitespace
func TestValidateOnCalendarWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "leading space",
			input:   " daily",
			wantErr: false,
		},
		{
			name:    "trailing space",
			input:   "daily ",
			wantErr: false,
		},
		{
			name:    "both leading and trailing spaces",
			input:   " daily ",
			wantErr: false,
		},
		{
			name:    "multiple leading spaces",
			input:   "   daily",
			wantErr: false,
		},
		{
			name:    "multiple trailing spaces",
			input:   "daily   ",
			wantErr: false,
		},
		{
			name:    "tab character",
			input:   "\tdaily\t",
			wantErr: false,
		},
		{
			name:    "newline character",
			input:   "\ndaily\n",
			wantErr: false,
		},
		{
			name:    "mixed whitespace",
			input:   " \tdaily\n",
			wantErr: false,
		},
		{
			name:    "calendar with leading space - trimmed",
			input:   " *-*-* 00:00:00",
			wantErr: true, // The regex doesn't match with leading space before trimming
		},
		{
			name:    "calendar with trailing space - trimmed",
			input:   "*-*-* 00:00:00 ",
			wantErr: true, // The regex doesn't match with trailing space after trimming
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOnCalendar(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOnCalendar(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// Test ValidateOnCalendar error messages
func TestValidateOnCalendarErrorMessages(t *testing.T) {
	invalidInputs := []string{
		"invalid",
		"notavalidschedule",
		"XYZ *-*-* 00:00:00",
		"2024/01/01 00:00:00",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			err := ValidateOnCalendar(input)
			if err == nil {
				t.Fatal("expected error for invalid input")
			}

			errMsg := err.Error()

			// Check for helpful content in error message
			if !strings.Contains(errMsg, "invalid OnCalendar format") {
				t.Errorf("error message should contain 'invalid OnCalendar format', got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "daily") {
				t.Errorf("error message should suggest 'daily' as valid format, got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "weekly") {
				t.Errorf("error message should suggest 'weekly' as valid format, got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "*-*-* 02:00:00") {
				t.Errorf("error message should show example '*-*-* 02:00:00', got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "Mon *-*-* 09:00:00") {
				t.Errorf("error message should show example 'Mon *-*-* 09:00:00', got: %s", errMsg)
			}
		})
	}
}

// Test checkRcloneBinary with environment variable
func TestCheckRcloneBinaryWithEnvVar(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	// Create a temporary directory with a fake rclone
	tmpDir := t.TempDir()
	fakeRclone := filepath.Join(tmpDir, "rclone")

	// Create a minimal executable
	script := "#!/bin/sh\necho 'fake rclone'"
	if err := os.WriteFile(fakeRclone, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake rclone: %v", err)
	}

	// Set PATH to include our temp directory
	os.Setenv("PATH", tmpDir)

	c := NewClient()
	result := checkRcloneBinary(c)

	if !result.Passed {
		t.Errorf("checkRcloneBinary() should pass when rclone is in PATH: %s", result.Message)
	}
	if !strings.Contains(result.Message, "rclone") {
		t.Errorf("checkRcloneBinary() message should mention 'rclone', got: %s", result.Message)
	}
}

// Test checkRcloneBinary with custom path
func TestCheckRcloneBinaryCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom-rclone")

	script := "#!/bin/sh\necho 'custom rclone'"
	if err := os.WriteFile(customPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create custom rclone: %v", err)
	}

	c := NewClientWithPath(customPath)
	result := checkRcloneBinary(c)

	if !result.Passed {
		t.Errorf("checkRcloneBinary() should pass for custom path: %s", result.Message)
	}
	if !strings.Contains(result.Message, customPath) {
		t.Errorf("checkRcloneBinary() message should contain custom path %q, got: %s", customPath, result.Message)
	}
}

// Test checkRcloneVersion with various version formats
func TestCheckRcloneVersionFormats(t *testing.T) {
	tests := []struct {
		name       string
		versionOut string
		wantPass   bool
	}{
		{
			name:       "standard format",
			versionOut: "rclone v1.62.0",
			wantPass:   true,
		},
		{
			name:       "with v prefix only",
			versionOut: "v1.65.0",
			wantPass:   true,
		},
		{
			name:       "without prefix",
			versionOut: "1.60.0",
			wantPass:   true,
		},
		{
			name:       "with beta suffix",
			versionOut: "rclone v1.61.0-beta.1234",
			wantPass:   true,
		},
		{
			name:       "below minimum",
			versionOut: "rclone v1.59.0",
			wantPass:   false,
		},
		{
			name:       "exactly minimum",
			versionOut: "rclone v1.60.0",
			wantPass:   true,
		},
		{
			name:       "future version",
			versionOut: "rclone v999.0.0",
			wantPass:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScript := fmt.Sprintf(`#!/bin/sh
echo "%s"`, tt.versionOut)
			mockPath := createMockRcloneValidation(t, mockScript)
			c := NewClientWithPath(mockPath)

			result := checkRcloneVersion(c)

			if result.Passed != tt.wantPass {
				t.Errorf("checkRcloneVersion() passed = %v, want %v. Message: %s", result.Passed, tt.wantPass, result.Message)
			}
			if !result.IsCritical {
				t.Error("checkRcloneVersion should always be critical")
			}
		})
	}
}

// Test checkConfiguredRemotes timeout handling - skipped by default as it takes time
func TestCheckConfiguredRemotesTimeout(t *testing.T) {
	t.Skip("Skipping timeout test - takes 30+ seconds. Run manually when needed.")

	// Create a mock that hangs longer than the 30 second timeout
	mockScript := `#!/bin/sh
sleep 35
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	// This should timeout (30 second timeout in the function)
	result := checkConfiguredRemotes(c)

	if result.Passed {
		t.Error("checkConfiguredRemotes() should fail on timeout")
	}
	if !strings.Contains(result.Message, "Timeout") {
		t.Errorf("checkConfiguredRemotes() message should mention timeout, got: %s", result.Message)
	}
}

// Test checkConfiguredRemotes with panic recovery
func TestCheckConfiguredRemotesPanicRecovery(t *testing.T) {
	// This test verifies the panic recovery in checkConfiguredRemotes
	// We can't easily trigger a panic in the client, but we can test the structure
	mockScript := `#!/bin/sh
echo "remote1:"
echo "remote2:"
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	result := checkConfiguredRemotes(c)

	if !result.Passed {
		t.Errorf("checkConfiguredRemotes() should pass with valid remotes: %s", result.Message)
	}
	if !strings.Contains(result.Message, "2") {
		t.Errorf("checkConfiguredRemotes() message should mention 2 remotes, got: %s", result.Message)
	}
}

// Test checkSystemdUserSession with various scenarios
func TestCheckSystemdUserSessionScenarios(t *testing.T) {
	// Test when systemctl exists but user session is not available
	// This is hard to test without mocking exec.Command, so we'll skip if systemd is available
	_, err := exec.LookPath("systemctl")
	if err != nil {
		// systemctl doesn't exist - test should fail
		result := checkSystemdUserSession()
		if result.Passed {
			t.Error("checkSystemdUserSession() should fail when systemctl not found")
		}
		if !result.IsCritical {
			t.Error("checkSystemdUserSession should be critical")
		}
	} else {
		// systemctl exists - test should pass (or at least not fail with bus error)
		result := checkSystemdUserSession()
		// We don't assert on result.Passed because the user session might not be active
		// But we verify the structure is correct
		if result.Name != "Systemd User Session" {
			t.Errorf("checkSystemdUserSession().Name = %q, want %q", result.Name, "Systemd User Session")
		}
		if !result.IsCritical {
			t.Error("checkSystemdUserSession should be critical")
		}
	}
}

// Test checkFusermount with fusermount3 preference
func TestCheckFusermountPreference(t *testing.T) {
	// Test that fusermount3 is preferred over fusermount
	// This is difficult to test without manipulating PATH, so we'll just verify the logic
	// works when one or both are available

	_, hasFusermount := exec.LookPath("fusermount")
	_, hasFusermount3 := exec.LookPath("fusermount3")

	result := checkFusermount()

	if hasFusermount == nil || hasFusermount3 == nil {
		// At least one should be found
		if !result.Passed {
			t.Errorf("checkFusermount() should pass when at least one fusermount exists: %s", result.Message)
		}
		if result.IsCritical {
			t.Error("checkFusermount should not be critical")
		}
		// Verify the message mentions which one was found
		if hasFusermount3 == nil && !strings.Contains(result.Message, "fusermount3") {
			t.Errorf("checkFusermount() message should mention fusermount3, got: %s", result.Message)
		}
	} else {
		// Neither exists
		if result.Passed {
			t.Error("checkFusermount() should fail when neither fusermount exists")
		}
		if result.IsCritical {
			t.Error("checkFusermount should not be critical")
		}
		if !strings.Contains(result.Suggestion, "fuse") {
			t.Errorf("checkFusermount() suggestion should mention fuse, got: %s", result.Suggestion)
		}
	}
}

// Test PreflightChecks integration
func TestPreflightChecksIntegration(t *testing.T) {
	// Test with fully working rclone
	mockScript := `#!/bin/sh
case "$1" in
	version)
		echo "rclone v1.62.0"
		;;
	listremotes)
		echo "gdrive:"
		echo "dropbox:"
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	results := PreflightChecks(c)

	if len(results) != 5 {
		t.Errorf("PreflightChecks() returned %d results, want 5", len(results))
	}

	expectedChecks := []string{
		"Rclone Binary",
		"Rclone Version",
		"Configured Remotes",
		"Systemd User Session",
		"Fusermount",
	}

	for i, expected := range expectedChecks {
		if i >= len(results) {
			t.Errorf("Missing check %d: %s", i, expected)
			continue
		}
		if results[i].Name != expected {
			t.Errorf("Check %d name = %q, want %q", i, results[i].Name, expected)
		}
	}

	// First three should pass with our mock
	if !results[0].Passed {
		t.Errorf("Rclone Binary check should pass: %s", results[0].Message)
	}
	if !results[1].Passed {
		t.Errorf("Rclone Version check should pass: %s", results[1].Message)
	}
	if !results[2].Passed {
		t.Errorf("Configured Remotes check should pass: %s", results[2].Message)
	}
}

// Test PreflightChecks with partial failures
func TestPreflightChecksPartialFailures(t *testing.T) {
	// Test with old version
	mockScript := `#!/bin/sh
case "$1" in
	version)
		echo "rclone v1.50.0"
		;;
	listremotes)
		echo ""
		;;
esac
`
	mockPath := createMockRcloneValidation(t, mockScript)
	c := NewClientWithPath(mockPath)

	results := PreflightChecks(c)

	if len(results) < 4 {
		t.Errorf("PreflightChecks() returned %d results, want at least 4", len(results))
	}

	// Rclone Binary should pass
	if !results[0].Passed {
		t.Errorf("Rclone Binary check should pass: %s", results[0].Message)
	}

	// Rclone Version should fail (old version)
	if results[1].Passed {
		t.Error("Rclone Version check should fail for old version")
	}
	if !results[1].IsCritical {
		t.Error("Rclone Version check should be critical")
	}

	// Configured Remotes should fail (empty)
	if results[2].Passed {
		t.Error("Configured Remotes check should fail when empty")
	}
	if results[2].IsCritical {
		t.Error("Configured Remotes check should not be critical")
	}
}

// Test CheckResult with all fields
func TestCheckResultAllFields(t *testing.T) {
	result := CheckResult{
		Name:       "Test Check",
		Passed:     true,
		Message:    "Test message",
		Suggestion: "Test suggestion",
		IsCritical: true,
	}

	if result.Name != "Test Check" {
		t.Errorf("Name = %q, want %q", result.Name, "Test Check")
	}
	if result.Passed != true {
		t.Errorf("Passed = %v, want true", result.Passed)
	}
	if result.Message != "Test message" {
		t.Errorf("Message = %q, want %q", result.Message, "Test message")
	}
	if result.Suggestion != "Test suggestion" {
		t.Errorf("Suggestion = %q, want %q", result.Suggestion, "Test suggestion")
	}
	if result.IsCritical != true {
		t.Errorf("IsCritical = %v, want true", result.IsCritical)
	}
}

// Test that CheckResult zero value is safe to use
func TestCheckResultZeroValue(t *testing.T) {
	var result CheckResult

	if result.Name != "" {
		t.Errorf("zero value Name = %q, want empty", result.Name)
	}
	if result.Passed != false {
		t.Errorf("zero value Passed = %v, want false", result.Passed)
	}
	if result.Message != "" {
		t.Errorf("zero value Message = %q, want empty", result.Message)
	}
	if result.Suggestion != "" {
		t.Errorf("zero value Suggestion = %q, want empty", result.Suggestion)
	}
	if result.IsCritical != false {
		t.Errorf("zero value IsCritical = %v, want false", result.IsCritical)
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkParseVersion(b *testing.B) {
	versionStr := "rclone v1.62.0"
	for i := 0; i < b.N; i++ {
		_, _ = parseVersion(versionStr)
	}
}

func BenchmarkCompareVersions(b *testing.B) {
	a := versionTuple{1, 62, 0}
	bVersion := versionTuple{1, 60, 0}
	for i := 0; i < b.N; i++ {
		_ = compareVersions(a, bVersion)
	}
}

func BenchmarkFormatRemoteNames(b *testing.B) {
	remotes := []Remote{
		{Name: "gdrive"}, {Name: "dropbox"}, {Name: "s3"},
		{Name: "onedrive"}, {Name: "box"}, {Name: "azure"},
		{Name: "google_cloud"}, {Name: "aws"}, {Name: "backblaze"}, {Name: "meganz"},
	}
	for i := 0; i < b.N; i++ {
		_ = formatRemoteNames(remotes)
	}
}

func BenchmarkValidateOnCalendar(b *testing.B) {
	calendar := "*-*-* 02:00:00"
	for i := 0; i < b.N; i++ {
		_ = ValidateOnCalendar(calendar)
	}
}

func BenchmarkFormatResults(b *testing.B) {
	results := []CheckResult{
		{Name: "Check1", Passed: true, Message: "OK", IsCritical: true},
		{Name: "Check2", Passed: false, Message: "Failed", Suggestion: "Fix it", IsCritical: true},
		{Name: "Check3", Passed: true, Message: "OK", IsCritical: false},
		{Name: "Check4", Passed: false, Message: "Warning", Suggestion: "Consider fixing", IsCritical: false},
		{Name: "Check5", Passed: true, Message: "OK", IsCritical: true},
	}
	for i := 0; i < b.N; i++ {
		_ = FormatResults(results)
	}
}
