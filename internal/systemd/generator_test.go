package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

// TestSanitizeName tests the sanitizeName function.
func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "gdrive",
			want:  "gdrive",
		},
		{
			name:  "name with spaces",
			input: "My Google Drive",
			want:  "my-google-drive",
		},
		{
			name:  "name with special characters",
			input: "gdrive@home!",
			want:  "gdrive-home",
		},
		{
			name:  "name with underscores",
			input: "my_remote",
			want:  "my_remote",
		},
		{
			name:  "name with multiple spaces",
			input: "my   remote",
			want:  "my-remote",
		},
		{
			name:  "name with leading dash",
			input: "-myremote",
			want:  "myremote",
		},
		{
			name:  "name with trailing dash",
			input: "myremote-",
			want:  "myremote",
		},
		{
			name:  "name with consecutive dashes",
			input: "my--remote",
			want:  "my-remote",
		},
		{
			name:  "name with mixed case",
			input: "MyRemote",
			want:  "myremote",
		},
		{
			name:  "name with numbers",
			input: "remote123",
			want:  "remote123",
		},
		{
			name:  "name with dots",
			input: "my.remote",
			want:  "my-remote",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "@#$%",
			want:  "",
		},
		{
			name:  "complex name",
			input: "My_Google.Drive @Work!",
			want:  "my_google-drive-work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestServiceName tests the Generator.ServiceName method.
func TestGenerator_ServiceName(t *testing.T) {
	// Create a generator with a temp directory
	tmpDir := t.TempDir()
	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	tests := []struct {
		name     string
		unitName string
		unitType string
		want     string
	}{
		{
			name:     "mount unit",
			unitName: "abc12345",
			unitType: "mount",
			want:     "rclone-mount-abc12345",
		},
		{
			name:     "sync unit",
			unitName: "def67890",
			unitType: "sync",
			want:     "rclone-sync-def67890",
		},
		{
			name:     "alphanumeric id",
			unitName: "a1b2c3d4",
			unitType: "sync",
			want:     "rclone-sync-a1b2c3d4",
		},
		{
			name:     "uppercase id",
			unitName: "ABC12345",
			unitType: "mount",
			want:     "rclone-mount-ABC12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ServiceName(tt.unitName, tt.unitType)
			if got != tt.want {
				t.Errorf("ServiceName(%q, %q) = %q, want %q", tt.unitName, tt.unitType, got, tt.want)
			}
		})
	}
}

// TestBuildMountOptions tests the buildMountOptions method.
func TestGenerator_BuildMountOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name     string
		opts     models.MountOptions
		contains []string
	}{
		{
			name:     "basic options",
			opts:     models.MountOptions{},
			contains: []string{"--config="},
		},
		{
			name: "with vfs cache",
			opts: models.MountOptions{
				VFSCacheMode: "full",
			},
			contains: []string{"--vfs-cache-mode=full"},
		},
		{
			name: "with allow other",
			opts: models.MountOptions{
				AllowOther: true,
			},
			contains: []string{"--allow-other"},
		},
		{
			name: "with buffer size",
			opts: models.MountOptions{
				BufferSize: "16M",
			},
			contains: []string{"--buffer-size=16M"},
		},
		{
			name: "with multiple options",
			opts: models.MountOptions{
				VFSCacheMode:   "writes",
				BufferSize:     "32M",
				AllowOther:     true,
				DirCacheTime:   "5m",
				ConnectTimeout: "30s",
				LogLevel:       "INFO",
			},
			contains: []string{
				"--vfs-cache-mode=writes",
				"--buffer-size=32M",
				"--allow-other",
				"--dir-cache-time=5m",
				"--connect-timeout=30s",
				"--log-level=INFO",
			},
		},
		{
			name: "with uid and gid",
			opts: models.MountOptions{
				UID: 1000,
				GID: 1000,
			},
			contains: []string{"--uid=1000", "--gid=1000"},
		},
		{
			name: "with umask",
			opts: models.MountOptions{
				Umask: "002",
			},
			contains: []string{"--umask=002"},
		},
		{
			name: "with read only",
			opts: models.MountOptions{
				ReadOnly: true,
			},
			contains: []string{"--read-only"},
		},
		{
			name: "with extra args",
			opts: models.MountOptions{
				ExtraArgs: "--poll-interval=15s",
			},
			contains: []string{"--poll-interval=15s"},
		},
		{
			name: "with custom config",
			opts: models.MountOptions{
				Config: "/custom/rclone.conf",
			},
			contains: []string{"--config=/custom/rclone.conf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.buildMountOptions(&tt.opts)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("buildMountOptions() missing expected %q in:\n%s", want, got)
				}
			}
		})
	}
}

// TestBuildSyncOptions tests the buildSyncOptions method.
func TestGenerator_BuildSyncOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name     string
		opts     models.SyncOptions
		contains []string
	}{
		{
			name:     "basic options",
			opts:     models.SyncOptions{},
			contains: []string{"--config=", "--create-empty-src-dirs"},
		},
		{
			name: "with transfers",
			opts: models.SyncOptions{
				Transfers: 4,
			},
			contains: []string{"--transfers=4"},
		},
		{
			name: "with checkers",
			opts: models.SyncOptions{
				Checkers: 8,
			},
			contains: []string{"--checkers=8"},
		},
		{
			name: "with bandwidth limit",
			opts: models.SyncOptions{
				BandwidthLimit: "10M",
			},
			contains: []string{"--bwlimit=10M"},
		},
		{
			name: "with include pattern",
			opts: models.SyncOptions{
				IncludePattern: "*.jpg",
			},
			contains: []string{"--include=*.jpg"},
		},
		{
			name: "with exclude pattern",
			opts: models.SyncOptions{
				ExcludePattern: "*.tmp",
			},
			contains: []string{"--exclude=*.tmp"},
		},
		{
			name: "with dry run",
			opts: models.SyncOptions{
				DryRun: true,
			},
			contains: []string{"--dry-run"},
		},
		{
			name: "with checksum",
			opts: models.SyncOptions{
				CheckSum: true,
			},
			contains: []string{"--checksum"},
		},
		{
			name: "with multiple options",
			opts: models.SyncOptions{
				Transfers:        4,
				Checkers:         8,
				BandwidthLimit:   "10M",
				LogLevel:         "DEBUG",
				DeleteExtraneous: true,
			},
			contains: []string{
				"--transfers=4",
				"--checkers=8",
				"--bwlimit=10M",
				"--log-level=DEBUG",
				"--delete-after",
			},
		},
		{
			name: "with max age",
			opts: models.SyncOptions{
				MaxAge: "30d",
			},
			contains: []string{"--max-age=30d"},
		},
		{
			name: "with min age",
			opts: models.SyncOptions{
				MinAge: "1d",
			},
			contains: []string{"--min-age=1d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.buildSyncOptions(&tt.opts)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("buildSyncOptions() missing expected %q in:\n%s", want, got)
				}
			}
		})
	}
}

// TestBuildTimerDirectives tests the buildTimerDirectives method.
func TestGenerator_BuildTimerDirectives(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name     string
		schedule models.ScheduleConfig
		contains []string
	}{
		{
			name:     "empty schedule defaults to daily",
			schedule: models.ScheduleConfig{},
			contains: []string{"OnCalendar=daily"},
		},
		{
			name: "timer with OnCalendar",
			schedule: models.ScheduleConfig{
				Type:       "timer",
				OnCalendar: "*-*-* 02:00:00",
			},
			contains: []string{"OnCalendar=*-*-* 02:00:00"},
		},
		{
			name: "timer with daily",
			schedule: models.ScheduleConfig{
				Type:       "timer",
				OnCalendar: "daily",
			},
			contains: []string{"OnCalendar=daily"},
		},
		{
			name: "onboot schedule",
			schedule: models.ScheduleConfig{
				Type:      "onboot",
				OnBootSec: "5min",
			},
			contains: []string{"OnBootSec=5min"},
		},
		{
			name: "with randomized delay",
			schedule: models.ScheduleConfig{
				Type:               "timer",
				OnCalendar:         "hourly",
				RandomizedDelaySec: "5m",
			},
			contains: []string{"OnCalendar=hourly", "RandomizedDelaySec=5m"},
		},
		{
			name: "with persistent",
			schedule: models.ScheduleConfig{
				Type:       "timer",
				OnCalendar: "daily",
				Persistent: true,
			},
			contains: []string{"OnCalendar=daily", "Persistent=true"},
		},
		{
			name: "with OnActiveSec",
			schedule: models.ScheduleConfig{
				Type:        "timer",
				OnCalendar:  "daily",
				OnActiveSec: "1h",
			},
			contains: []string{"OnUnitActiveSec=1h"},
		},
		{
			name: "complex schedule",
			schedule: models.ScheduleConfig{
				Type:               "timer",
				OnCalendar:         "*-*-* 02:00:00",
				RandomizedDelaySec: "10m",
				Persistent:         true,
			},
			contains: []string{
				"OnCalendar=*-*-* 02:00:00",
				"RandomizedDelaySec=10m",
				"Persistent=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.buildTimerDirectives(&tt.schedule)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("buildTimerDirectives() missing expected %q in:\n%s", want, got)
				}
			}
		})
	}
}

// TestGenerateMountService tests the GenerateMountService method.
func TestGenerator_GenerateMountService(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	mount := &models.MountConfig{
		ID:          "a1b2c3d4",
		Name:        "gdrive",
		Remote:      "gdrive:",
		RemotePath:  "/",
		MountPoint:  "/mnt/gdrive",
		Description: "Google Drive mount",
	}

	content, err := g.GenerateMountService(mount)
	if err != nil {
		t.Fatalf("GenerateMountService() error = %v", err)
	}

	// Verify the content contains expected sections
	expectedSections := []string{
		"[Unit]",
		"Description=Rclone mount: gdrive",
		"[Service]",
		"Type=notify",
		"ExecStart=/usr/bin/rclone mount",
		"gdrive:/",
		"/mnt/gdrive",
		"[Install]",
		"WantedBy=default.target",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("GenerateMountService() missing expected section %q", section)
		}
	}
}

// TestGenerateSyncService tests the GenerateSyncService method.
func TestGenerator_GenerateSyncService(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	job := &models.SyncJobConfig{
		ID:          "e5f6g7h8",
		Name:        "backup-photos",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup/Photos",
		SyncOptions: models.SyncOptions{
			Direction: "sync",
		},
	}

	content, err := g.GenerateSyncService(job)
	if err != nil {
		t.Fatalf("GenerateSyncService() error = %v", err)
	}

	// Verify the content contains expected sections
	expectedSections := []string{
		"[Unit]",
		"Description=Rclone sync: backup-photos",
		"[Service]",
		"Type=oneshot",
		"ExecStart=/usr/bin/rclone sync",
		"gdrive:/Photos",
		"/home/user/Backup/Photos",
		"[Install]",
		"WantedBy=default.target",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("GenerateSyncService() missing expected section %q", section)
		}
	}
}

// TestGenerateSyncTimer tests the GenerateSyncTimer method.
func TestGenerator_GenerateSyncTimer(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name     string
		job      *models.SyncJobConfig
		contains []string
	}{
		{
			name: "daily timer",
			job: &models.SyncJobConfig{
				ID:   "i9j0k1l2",
				Name: "backup",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
				},
			},
			contains: []string{
				"[Unit]",
				"Description=Timer for rclone sync: backup",
				"[Timer]",
				"OnCalendar=daily",
				"WantedBy=timers.target",
			},
		},
		{
			name: "hourly timer",
			job: &models.SyncJobConfig{
				ID:   "m3n4o5p6",
				Name: "frequent-backup",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "hourly",
				},
			},
			contains: []string{"OnCalendar=hourly"},
		},
		{
			name: "timer with persistent",
			job: &models.SyncJobConfig{
				ID:   "q7r8s9t0",
				Name: "persistent-backup",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
					Persistent: true,
				},
			},
			contains: []string{"Persistent=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateSyncTimer(tt.job)
			if err != nil {
				t.Fatalf("GenerateSyncTimer() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(content, want) {
					t.Errorf("GenerateSyncTimer() missing expected %q in:\n%s", want, content)
				}
			}
		})
	}
}

// TestWriteUnitFile tests the WriteUnitFile method.
func TestGenerator_WriteUnitFile(t *testing.T) {
	tmpDir := t.TempDir()
	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	filename := "test.service"
	content := "[Unit]\nDescription=Test\n"

	err := g.WriteUnitFile(filename, content)
	if err != nil {
		t.Fatalf("WriteUnitFile() error = %v", err)
	}

	// Verify file was created
	path := filepath.Join(tmpDir, filename)
	readContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("WriteUnitFile() wrote %q, want %q", string(readContent), content)
	}
}

// TestWriteUnitFileCreatesDirectory tests that WriteUnitFile creates the directory if needed.
func TestGenerator_WriteUnitFileCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a subdirectory that doesn't exist
	systemdDir := filepath.Join(tmpDir, "systemd", "user")

	g := &Generator{
		systemdDir: systemdDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	filename := "test.service"
	content := "[Unit]\nDescription=Test\n"

	err := g.WriteUnitFile(filename, content)
	if err != nil {
		t.Fatalf("WriteUnitFile() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(systemdDir); os.IsNotExist(err) {
		t.Error("WriteUnitFile() did not create systemd directory")
	}
}

// TestRemoveUnit tests the RemoveUnit method.
func TestGenerator_RemoveUnit(t *testing.T) {
	tmpDir := t.TempDir()
	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	// Create a file to remove
	filename := "to-remove.service"
	path := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Remove it
	err := g.RemoveUnit(filename)
	if err != nil {
		t.Fatalf("RemoveUnit() error = %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("RemoveUnit() did not remove the file")
	}
}

// TestRemoveUnitNonExistent tests RemoveUnit on a non-existent file.
func TestGenerator_RemoveUnitNonExistent(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	// Should not error on non-existent file
	err := g.RemoveUnit("nonexistent.service")
	if err != nil {
		t.Errorf("RemoveUnit() error = %v, want nil for non-existent file", err)
	}
}

// TestMountUnitData tests that MountUnitData struct is properly populated.
func TestMountUnitData(t *testing.T) {
	data := MountUnitData{
		Name:         "test-mount",
		Remote:       "gdrive:",
		RemotePath:   "/Photos",
		MountPoint:   "/mnt/gdrive",
		ConfigPath:   "/home/user/.config/rclone/rclone.conf",
		MountOptions: "--vfs-cache-mode=full",
		LogLevel:     "INFO",
		LogPath:      "/var/log/rclone.log",
		RclonePath:   "/usr/bin/rclone",
	}

	if data.Name != "test-mount" {
		t.Errorf("MountUnitData.Name = %q, want %q", data.Name, "test-mount")
	}
	if data.Remote != "gdrive:" {
		t.Errorf("MountUnitData.Remote = %q, want %q", data.Remote, "gdrive:")
	}
}

// TestSyncUnitData tests that SyncUnitData struct is properly populated.
func TestSyncUnitData(t *testing.T) {
	data := SyncUnitData{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup",
		Direction:   "sync",
		ConfigPath:  "/home/user/.config/rclone/rclone.conf",
		SyncOptions: "--transfers=4",
		LogLevel:    "DEBUG",
		LogPath:     "/var/log/rclone-sync.log",
		RclonePath:  "/usr/bin/rclone",
	}

	if data.Name != "test-sync" {
		t.Errorf("SyncUnitData.Name = %q, want %q", data.Name, "test-sync")
	}
	if data.Direction != "sync" {
		t.Errorf("SyncUnitData.Direction = %q, want %q", data.Direction, "sync")
	}
}

// TestTimerUnitData tests that TimerUnitData struct is properly populated.
func TestTimerUnitData(t *testing.T) {
	data := TimerUnitData{
		Name:            "test-timer",
		TimerDirectives: "OnCalendar=daily",
	}

	if data.Name != "test-timer" {
		t.Errorf("TimerUnitData.Name = %q, want %q", data.Name, "test-timer")
	}
	if data.TimerDirectives != "OnCalendar=daily" {
		t.Errorf("TimerUnitData.TimerDirectives = %q, want %q", data.TimerDirectives, "OnCalendar=daily")
	}
}

// TestUserSystemdDirConstant tests that the constant is correctly defined.
func TestUserSystemdDirConstant(t *testing.T) {
	expected := ".config/systemd/user"
	if UserSystemdDir != expected {
		t.Errorf("UserSystemdDir = %q, want %q", UserSystemdDir, expected)
	}
}

// TestMountServiceTemplateContainsRequiredFields tests that the mount template has required fields.
func TestMountServiceTemplateContainsRequiredFields(t *testing.T) {
	requiredFields := []string{
		"{{.Name}}",
		"{{.Remote}}",
		"{{.RemotePath}}",
		"{{.MountPoint}}",
		"{{.MountOptions}}",
		"{{.RclonePath}}",
		"[Unit]",
		"[Service]",
		"[Install]",
		"Type=notify",
		"Restart=",
	}

	for _, field := range requiredFields {
		if !strings.Contains(MountServiceTemplate, field) {
			t.Errorf("MountServiceTemplate missing required field: %s", field)
		}
	}
}

// TestSyncServiceTemplateContainsRequiredFields tests that the sync template has required fields.
func TestSyncServiceTemplateContainsRequiredFields(t *testing.T) {
	requiredFields := []string{
		"{{.Name}}",
		"{{.Source}}",
		"{{.Destination}}",
		"{{.Direction}}",
		"{{.SyncOptions}}",
		"{{.RclonePath}}",
		"[Unit]",
		"[Service]",
		"[Install]",
		"Type=oneshot",
	}

	for _, field := range requiredFields {
		if !strings.Contains(SyncServiceTemplate, field) {
			t.Errorf("SyncServiceTemplate missing required field: %s", field)
		}
	}
}

// TestSyncTimerTemplateContainsRequiredFields tests that the timer template has required fields.
func TestSyncTimerTemplateContainsRequiredFields(t *testing.T) {
	requiredFields := []string{
		"{{.Name}}",
		"{{.TimerDirectives}}",
		"[Unit]",
		"[Timer]",
		"[Install]",
		"WantedBy=timers.target",
	}

	for _, field := range requiredFields {
		if !strings.Contains(SyncTimerTemplate, field) {
			t.Errorf("SyncTimerTemplate missing required field: %s", field)
		}
	}
}

// TestGetSystemdDir tests the GetSystemdDir method.
func TestGenerator_GetSystemdDir(t *testing.T) {
	expected := "/test/systemd/dir"
	g := &Generator{
		systemdDir: expected,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     "/tmp",
	}

	got := g.GetSystemdDir()
	if got != expected {
		t.Errorf("GetSystemdDir() = %q, want %q", got, expected)
	}
}

// TestGenerateMountServiceWithMountOptions tests mount service generation with various mount options.
func TestGenerator_GenerateMountServiceWithMountOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	mount := &models.MountConfig{
		ID:         "u1v2w3x4",
		Name:       "gdrive",
		Remote:     "gdrive:",
		RemotePath: "/",
		MountPoint: "/mnt/gdrive",
		MountOptions: models.MountOptions{
			VFSCacheMode: "full",
			BufferSize:   "16M",
			AllowOther:   true,
			LogLevel:     "INFO",
		},
	}

	content, err := g.GenerateMountService(mount)
	if err != nil {
		t.Fatalf("GenerateMountService() error = %v", err)
	}

	expectedOptions := []string{
		"--vfs-cache-mode=full",
		"--buffer-size=16M",
		"--allow-other",
		"--log-level=INFO",
	}

	for _, opt := range expectedOptions {
		if !strings.Contains(content, opt) {
			t.Errorf("GenerateMountService() missing expected option %q", opt)
		}
	}
}

// TestGenerateSyncServiceWithSyncOptions tests sync service generation with various sync options.
func TestGenerator_GenerateSyncServiceWithSyncOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	job := &models.SyncJobConfig{
		ID:          "y5z6a7b8",
		Name:        "backup",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup/Photos",
		SyncOptions: models.SyncOptions{
			Direction:      "sync",
			Transfers:      4,
			Checkers:       8,
			DryRun:         true,
			BandwidthLimit: "10M",
		},
	}

	content, err := g.GenerateSyncService(job)
	if err != nil {
		t.Fatalf("GenerateSyncService() error = %v", err)
	}

	expectedOptions := []string{
		"--transfers=4",
		"--checkers=8",
		"--dry-run",
		"--bwlimit=10M",
	}

	for _, opt := range expectedOptions {
		if !strings.Contains(content, opt) {
			t.Errorf("GenerateSyncService() missing expected option %q", opt)
		}
	}
}

// TestNewGenerator tests the NewGenerator function.
func TestNewGenerator(t *testing.T) {
	g, err := NewGenerator()
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}
	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}
	if g.systemdDir == "" {
		t.Error("NewGenerator() systemdDir is empty")
	}
	if g.rclonePath == "" {
		t.Error("NewGenerator() rclonePath is empty")
	}
	if g.configPath == "" {
		t.Error("NewGenerator() configPath is empty")
	}
	if g.logDir == "" {
		t.Error("NewGenerator() logDir is empty")
	}
}

// TestWriteMountService tests the WriteMountService method.
func TestGenerator_WriteMountService(t *testing.T) {
	tmpDir := t.TempDir()
	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	mount := &models.MountConfig{
		ID:          "c9d0e1f2",
		Name:        "test-mount",
		Remote:      "gdrive:",
		RemotePath:  "/",
		MountPoint:  "/mnt/gdrive",
		Description: "Test mount",
	}

	path, err := g.WriteMountService(mount)
	if err != nil {
		t.Fatalf("WriteMountService() error = %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("WriteMountService() returned relative path %q", path)
	}

	if !strings.HasSuffix(path, ".service") {
		t.Errorf("WriteMountService() returned path without .service suffix: %q", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if len(content) == 0 {
		t.Error("WriteMountService() wrote empty file")
	}
}

// TestWriteSyncUnits tests the WriteSyncUnits method.
func TestGenerator_WriteSyncUnits(t *testing.T) {
	tmpDir := t.TempDir()
	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}

	tests := []struct {
		name        string
		job         *models.SyncJobConfig
		wantTimer   bool
		wantService bool
	}{
		{
			name: "timer schedule",
			job: &models.SyncJobConfig{
				ID:          "g3h4i5j6",
				Name:        "backup-timer",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup/Photos",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
				},
			},
			wantTimer:   true,
			wantService: true,
		},
		{
			name: "onboot schedule",
			job: &models.SyncJobConfig{
				ID:          "k7l8m9n0",
				Name:        "backup-onboot",
				Source:      "gdrive:/Documents",
				Destination: "/home/user/Backup/Documents",
				Schedule: models.ScheduleConfig{
					Type:      "onboot",
					OnBootSec: "5min",
				},
			},
			wantTimer:   true,
			wantService: true,
		},
		{
			name: "manual schedule",
			job: &models.SyncJobConfig{
				ID:          "o1p2q3r4",
				Name:        "backup-manual",
				Source:      "gdrive:/Manual",
				Destination: "/home/user/Backup/Manual",
				Schedule: models.ScheduleConfig{
					Type: "manual",
				},
			},
			wantTimer:   false,
			wantService: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			servicePath, timerPath, err := g.WriteSyncUnits(tt.job)
			if err != nil {
				t.Fatalf("WriteSyncUnits() error = %v", err)
			}

			if tt.wantService {
				if servicePath == "" {
					t.Error("WriteSyncUnits() servicePath is empty")
				}
				if _, err := os.Stat(servicePath); os.IsNotExist(err) {
					t.Errorf("WriteSyncUnits() service file not created at %q", servicePath)
				}
			}

			if tt.wantTimer {
				if timerPath == "" {
					t.Error("WriteSyncUnits() timerPath is empty for timer schedule")
				}
				if _, err := os.Stat(timerPath); os.IsNotExist(err) {
					t.Errorf("WriteSyncUnits() timer file not created at %q", timerPath)
				}
			} else {
				if timerPath != "" {
					t.Errorf("WriteSyncUnits() timerPath should be empty for manual schedule, got %q", timerPath)
				}
			}
		})
	}
}

// TestExpandPath tests the expandPath function.
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "path with tilde",
			input:    "~/Documents",
			contains: "Documents",
		},
		{
			name:     "absolute path",
			input:    "/mnt/gdrive",
			contains: "/mnt/gdrive",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			contains: "relative/path",
		},
		{
			name:     "empty path",
			input:    "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.input)
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("expandPath(%q) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

// TestExpandPath_TildeExpansion tests that tilde expansion works correctly.
func TestExpandPath_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	result := expandPath("~/test")
	expected := filepath.Join(home, "test")
	if result != expected {
		t.Errorf("expandPath(\"~/test\") = %q, want %q", result, expected)
	}
}

// TestGetUserSystemdPath tests the GetUserSystemdPath function.
func TestGetUserSystemdPath(t *testing.T) {
	path, err := GetUserSystemdPath()
	if err != nil {
		t.Fatalf("GetUserSystemdPath() error = %v", err)
	}
	if path == "" {
		t.Error("GetUserSystemdPath() returned empty path")
	}
	if !strings.Contains(path, "systemd") {
		t.Errorf("GetUserSystemdPath() = %q, should contain 'systemd'", path)
	}
}

// TestGetRcloneConfigPath tests the getRcloneConfigPath function.
func TestGetRcloneConfigPath(t *testing.T) {
	path := getRcloneConfigPath()
	if path == "" {
		t.Error("getRcloneConfigPath() returned empty path")
	}
	if !strings.Contains(path, "rclone") {
		t.Errorf("getRcloneConfigPath() = %q, should contain 'rclone'", path)
	}
}

// TestGetRcloneConfigPath_WithEnv tests getRcloneConfigPath with RCLONE_CONFIG env var.
func TestGetRcloneConfigPath_WithEnv(t *testing.T) {
	originalEnv := os.Getenv("RCLONE_CONFIG")
	defer os.Setenv("RCLONE_CONFIG", originalEnv)

	os.Setenv("RCLONE_CONFIG", "/custom/path/rclone.conf")
	path := getRcloneConfigPath()
	if path != "/custom/path/rclone.conf" {
		t.Errorf("getRcloneConfigPath() = %q, want %q", path, "/custom/path/rclone.conf")
	}
}

// TestGetLogDir tests the getLogDir function.
func TestGetLogDir(t *testing.T) {
	dir, err := getLogDir()
	if err != nil {
		t.Fatalf("getLogDir() error = %v", err)
	}
	if dir == "" {
		t.Error("getLogDir() returned empty path")
	}
	if !strings.Contains(dir, "rclone-mount-sync") {
		t.Errorf("getLogDir() = %q, should contain 'rclone-mount-sync'", dir)
	}
}

// TestGetLogDir_WithXdgStateHome tests getLogDir with XDG_STATE_HOME env var.
func TestGetLogDir_WithXdgStateHome(t *testing.T) {
	originalEnv := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", originalEnv)

	tmpDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", tmpDir)

	dir, err := getLogDir()
	if err != nil {
		t.Fatalf("getLogDir() error = %v", err)
	}
	if !strings.HasPrefix(dir, tmpDir) {
		t.Errorf("getLogDir() = %q, should start with %q", dir, tmpDir)
	}
}

// TestGenerateMountService_EdgeCases tests mount service generation edge cases.
func TestGenerator_GenerateMountService_EdgeCases(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name    string
		mount   *models.MountConfig
		wantErr bool
	}{
		{
			name: "empty name",
			mount: &models.MountConfig{
				ID:         "s5t6u7v8",
				Name:       "",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/mnt/gdrive",
			},
			wantErr: false,
		},
		{
			name: "special characters in name",
			mount: &models.MountConfig{
				ID:         "w9x0y1z2",
				Name:       "My-Google.Drive@Work!",
				Remote:     "gdrive:",
				RemotePath: "/Work",
				MountPoint: "/mnt/work",
			},
			wantErr: false,
		},
		{
			name: "path with tilde",
			mount: &models.MountConfig{
				ID:         "a3b4c5d6",
				Name:       "home-mount",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "~/gdrive",
			},
			wantErr: false,
		},
		{
			name: "empty remote path",
			mount: &models.MountConfig{
				ID:         "e7f8g9h0",
				Name:       "empty-remote-path",
				Remote:     "gdrive:",
				RemotePath: "",
				MountPoint: "/mnt/gdrive",
			},
			wantErr: false,
		},
		{
			name: "all mount options",
			mount: &models.MountConfig{
				ID:         "i1j2k3l4",
				Name:       "full-options",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/mnt/full",
				MountOptions: models.MountOptions{
					VFSCacheMode:     "full",
					VFSCacheMaxAge:   "24h",
					VFSCacheMaxSize:  "10G",
					VFSReadChunkSize: "64M",
					VFSWriteBack:     "5s",
					BufferSize:       "16M",
					DirCacheTime:     "5m",
					AllowOther:       true,
					AllowRoot:        true,
					Umask:            "002",
					UID:              1000,
					GID:              1000,
					NoModTime:        true,
					NoChecksum:       true,
					ReadOnly:         true,
					ConnectTimeout:   "30s",
					Timeout:          "1m",
					LogLevel:         "DEBUG",
					Config:           "/custom/rclone.conf",
					ExtraArgs:        "--poll-interval=15s",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateMountService(tt.mount)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMountService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(content) == 0 {
				t.Error("GenerateMountService() returned empty content")
			}
		})
	}
}

// TestGenerateSyncService_EdgeCases tests sync service generation edge cases.
func TestGenerator_GenerateSyncService_EdgeCases(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name    string
		job     *models.SyncJobConfig
		wantErr bool
	}{
		{
			name: "empty direction defaults to sync",
			job: &models.SyncJobConfig{
				ID:          "m5n6o7p8",
				Name:        "default-direction",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup",
				SyncOptions: models.SyncOptions{
					Direction: "",
				},
			},
			wantErr: false,
		},
		{
			name: "copy direction",
			job: &models.SyncJobConfig{
				ID:          "q9r0s1t2",
				Name:        "copy-job",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup",
				SyncOptions: models.SyncOptions{
					Direction: "copy",
				},
			},
			wantErr: false,
		},
		{
			name: "move direction",
			job: &models.SyncJobConfig{
				ID:          "u3v4w5x6",
				Name:        "move-job",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup",
				SyncOptions: models.SyncOptions{
					Direction: "move",
				},
			},
			wantErr: false,
		},
		{
			name: "all sync options",
			job: &models.SyncJobConfig{
				ID:          "y7z8a9b0",
				Name:        "full-options",
				Source:      "gdrive:/Full",
				Destination: "/home/user/Full",
				SyncOptions: models.SyncOptions{
					Direction:        "sync",
					DeleteExtraneous: true,
					DeleteAfter:      true,
					IncludePattern:   "*.jpg",
					ExcludePattern:   "*.tmp",
					MaxAge:           "30d",
					MinAge:           "1d",
					Transfers:        4,
					Checkers:         8,
					BandwidthLimit:   "10M",
					CheckSum:         true,
					DryRun:           true,
					LogLevel:         "DEBUG",
					Config:           "/custom/rclone.conf",
					ExtraArgs:        "--stats=1m",
				},
			},
			wantErr: false,
		},
		{
			name: "path with tilde",
			job: &models.SyncJobConfig{
				ID:          "c1d2e3f4",
				Name:        "tilde-path",
				Source:      "gdrive:/Docs",
				Destination: "~/Backup/Docs",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateSyncService(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSyncService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(content) == 0 {
				t.Error("GenerateSyncService() returned empty content")
			}
		})
	}
}

// TestGenerateSyncTimer_EdgeCases tests timer generation edge cases.
func TestGenerator_GenerateSyncTimer_EdgeCases(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}

	tests := []struct {
		name     string
		job      *models.SyncJobConfig
		contains []string
	}{
		{
			name: "weekly schedule",
			job: &models.SyncJobConfig{
				ID:   "g5h6i7j8",
				Name: "weekly-backup",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "weekly",
				},
			},
			contains: []string{"OnCalendar=weekly"},
		},
		{
			name: "multiple OnCalendar expressions",
			job: &models.SyncJobConfig{
				ID:   "k9l0m1n2",
				Name: "multi-schedule",
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "*-*-* 00,06,12,18:00:00",
				},
			},
			contains: []string{"OnCalendar=*-*-* 00,06,12,18:00:00"},
		},
		{
			name: "onboot with onactive",
			job: &models.SyncJobConfig{
				ID:   "o3p4q5r6",
				Name: "boot-and-repeat",
				Schedule: models.ScheduleConfig{
					Type:        "onboot",
					OnBootSec:   "5min",
					OnActiveSec: "1h",
				},
			},
			contains: []string{"OnBootSec=5min", "OnUnitActiveSec=1h"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateSyncTimer(tt.job)
			if err != nil {
				t.Fatalf("GenerateSyncTimer() error = %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(content, want) {
					t.Errorf("GenerateSyncTimer() missing expected %q in:\n%s", want, content)
				}
			}
		})
	}
}

// TestBuildMountOptions_AllOptions tests all mount options are included.
func TestGenerator_BuildMountOptions_AllOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.MountOptions{
		VFSCacheMode:     "full",
		VFSCacheMaxAge:   "24h",
		VFSCacheMaxSize:  "10G",
		VFSReadChunkSize: "64M",
		VFSWriteBack:     "5s",
		BufferSize:       "16M",
		DirCacheTime:     "5m",
		AllowOther:       true,
		AllowRoot:        true,
		Umask:            "002",
		UID:              1000,
		GID:              1000,
		NoModTime:        true,
		NoChecksum:       true,
		ReadOnly:         true,
		ConnectTimeout:   "30s",
		Timeout:          "1m",
		LogLevel:         "DEBUG",
		Config:           "/custom/config.conf",
		ExtraArgs:        "--custom-arg",
	}

	result := g.buildMountOptions(opts)

	expectedContains := []string{
		"--vfs-cache-mode=full",
		"--vfs-cache-max-age=24h",
		"--vfs-cache-max-size=10G",
		"--vfs-read-chunk-size=64M",
		"--vfs-write-back=5s",
		"--buffer-size=16M",
		"--dir-cache-time=5m",
		"--allow-other",
		"--allow-root",
		"--umask=002",
		"--uid=1000",
		"--gid=1000",
		"--no-modtime",
		"--no-checksum",
		"--read-only",
		"--connect-timeout=30s",
		"--timeout=1m",
		"--log-level=DEBUG",
		"--config=/custom/config.conf",
		"--custom-arg",
	}

	for _, want := range expectedContains {
		if !strings.Contains(result, want) {
			t.Errorf("buildMountOptions() missing expected %q", want)
		}
	}
}

// TestBuildSyncOptions_AllOptions tests all sync options are included.
func TestGenerator_BuildSyncOptions_AllOptions(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.SyncOptions{
		Direction:        "sync",
		DeleteExtraneous: true,
		DeleteAfter:      false,
		IncludePattern:   "*.jpg,*.png",
		ExcludePattern:   "*.tmp",
		MaxAge:           "30d",
		MinAge:           "1d",
		Transfers:        4,
		Checkers:         8,
		BandwidthLimit:   "10M",
		CheckSum:         true,
		DryRun:           true,
		LogLevel:         "DEBUG",
		Config:           "/custom/config.conf",
		ExtraArgs:        "--stats=1m",
	}

	result := g.buildSyncOptions(opts)

	expectedContains := []string{
		"--delete-after",
		"--include=*.jpg,*.png",
		"--exclude=*.tmp",
		"--max-age=30d",
		"--min-age=1d",
		"--transfers=4",
		"--checkers=8",
		"--bwlimit=10M",
		"--checksum",
		"--dry-run",
		"--log-level=DEBUG",
		"--config=/custom/config.conf",
		"--create-empty-src-dirs",
		"--stats=1m",
	}

	for _, want := range expectedContains {
		if !strings.Contains(result, want) {
			t.Errorf("buildSyncOptions() missing expected %q", want)
		}
	}
}

// TestBuildSyncOptions_CustomConfig tests that custom config is used when specified.
func TestGenerator_BuildSyncOptions_CustomConfig(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.SyncOptions{
		Config: "/custom/rclone.conf",
	}

	result := g.buildSyncOptions(opts)
	if !strings.Contains(result, "--config=/custom/rclone.conf") {
		t.Errorf("buildSyncOptions() should use custom config, got: %s", result)
	}
}

// TestBuildMountOptions_CustomConfig tests that custom config is used when specified.
func TestGenerator_BuildMountOptions_CustomConfig(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.MountOptions{
		Config: "/custom/rclone.conf",
	}

	result := g.buildMountOptions(opts)
	if !strings.Contains(result, "--config=/custom/rclone.conf") {
		t.Errorf("buildMountOptions() should use custom config, got: %s", result)
	}
}

// TestBuildMountOptions_DefaultConfig tests that default config is used when not specified.
func TestGenerator_BuildMountOptions_DefaultConfig(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.MountOptions{}

	result := g.buildMountOptions(opts)
	if !strings.Contains(result, "--config=/default/config.conf") {
		t.Errorf("buildMountOptions() should use default config, got: %s", result)
	}
}

// TestBuildSyncOptions_DefaultConfig tests that default config is used when not specified.
func TestGenerator_BuildSyncOptions_DefaultConfig(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/default/config.conf",
		logDir:     t.TempDir(),
	}

	opts := &models.SyncOptions{}

	result := g.buildSyncOptions(opts)
	if !strings.Contains(result, "--config=/default/config.conf") {
		t.Errorf("buildSyncOptions() should use default config, got: %s", result)
	}
}
