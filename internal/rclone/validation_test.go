package rclone

import (
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
