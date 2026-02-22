package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

func TestNewReconciler(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()

	r := NewReconciler(g, m)
	if r == nil {
		t.Fatal("NewReconciler() returned nil")
	}
	if r.generator != g {
		t.Error("NewReconciler() generator not set correctly")
	}
	if r.manager == nil {
		t.Error("NewReconciler() manager not set correctly")
	}
}

func TestReconciler_ScanForOrphans(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		validMounts map[string]bool
		validSyncs  map[string]bool
		wantOrphans int
		wantErr     bool
	}{
		{
			name:        "empty directory",
			files:       map[string]string{},
			validMounts: map[string]bool{},
			validSyncs:  map[string]bool{},
			wantOrphans: 0,
			wantErr:     false,
		},
		{
			name: "no orphans",
			files: map[string]string{
				"rclone-mount-a1b2c3d4.service": "[Unit]\nDescription=Test",
				"rclone-sync-e5f6g7h8.service":  "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{"a1b2c3d4": true},
			validSyncs:  map[string]bool{"e5f6g7h8": true},
			wantOrphans: 0,
			wantErr:     false,
		},
		{
			name: "orphan mount",
			files: map[string]string{
				"rclone-mount-xyz12345.service": "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{"a1b2c3d4": true},
			validSyncs:  map[string]bool{},
			wantOrphans: 1,
			wantErr:     false,
		},
		{
			name: "orphan sync",
			files: map[string]string{
				"rclone-sync-abc12345.service": "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{},
			validSyncs:  map[string]bool{"e5f6g7h8": true},
			wantOrphans: 1,
			wantErr:     false,
		},
		{
			name: "skip timer files",
			files: map[string]string{
				"rclone-sync-a1b2c3d4.service": "[Unit]\nDescription=Test",
				"rclone-sync-a1b2c3d4.timer":   "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{},
			validSyncs:  map[string]bool{"a1b2c3d4": true},
			wantOrphans: 0,
			wantErr:     false,
		},
		{
			name: "skip non-rclone files",
			files: map[string]string{
				"other-service.service": "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{},
			validSyncs:  map[string]bool{},
			wantOrphans: 0,
			wantErr:     false,
		},
		{
			name: "legacy name-based unit",
			files: map[string]string{
				"rclone-mount-my-gdrive.service": "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{},
			validSyncs:  map[string]bool{},
			wantOrphans: 1,
			wantErr:     false,
		},
		{
			name: "multiple orphans",
			files: map[string]string{
				"rclone-mount-aaa11111.service": "[Unit]\nDescription=Test",
				"rclone-mount-bbb22222.service": "[Unit]\nDescription=Test",
				"rclone-sync-ccc33333.service":  "[Unit]\nDescription=Test",
			},
			validMounts: map[string]bool{"a1b2c3d4": true},
			validSyncs:  map[string]bool{"e5f6g7h8": true},
			wantOrphans: 3,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for name, content := range tt.files {
				path := filepath.Join(tmpDir, name)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			g := &Generator{
				systemdDir: tmpDir,
				rclonePath: "/usr/bin/rclone",
				configPath: "/home/user/.config/rclone/rclone.conf",
				logDir:     tmpDir,
			}
			m := NewManager()
			r := NewReconciler(g, m)

			result, err := r.ScanForOrphans(tt.validMounts, tt.validSyncs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScanForOrphans() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result.OrphanedUnits) != tt.wantOrphans {
				t.Errorf("ScanForOrphans() found %d orphans, want %d", len(result.OrphanedUnits), tt.wantOrphans)
			}
		})
	}
}

func TestReconciler_ScanForOrphans_NonexistentDir(t *testing.T) {
	g := &Generator{
		systemdDir: "/nonexistent/path/that/does/not/exist",
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	result, err := r.ScanForOrphans(map[string]bool{}, map[string]bool{})
	if err != nil {
		t.Errorf("ScanForOrphans() error = %v, want nil for nonexistent dir", err)
	}
	if len(result.OrphanedUnits) != 0 {
		t.Errorf("ScanForOrphans() found orphans for nonexistent dir")
	}
}

func TestReconciler_parseUnitFile(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	tests := []struct {
		name       string
		filename   string
		wantID     string
		wantType   string
		wantLegacy bool
	}{
		{
			name:       "mount unit with id",
			filename:   "rclone-mount-a1b2c3d4.service",
			wantID:     "a1b2c3d4",
			wantType:   "mount",
			wantLegacy: false,
		},
		{
			name:       "sync unit with id",
			filename:   "rclone-sync-e5f6g7h8.service",
			wantID:     "e5f6g7h8",
			wantType:   "sync",
			wantLegacy: false,
		},
		{
			name:       "legacy mount with name",
			filename:   "rclone-mount-my-gdrive.service",
			wantID:     "my-gdrive",
			wantType:   "mount",
			wantLegacy: true,
		},
		{
			name:       "legacy sync with name",
			filename:   "rclone-sync-backup-photos.service",
			wantID:     "backup-photos",
			wantType:   "sync",
			wantLegacy: true,
		},
		{
			name:       "unknown prefix",
			filename:   "other-service.service",
			wantID:     "",
			wantType:   "",
			wantLegacy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, unitType, isLegacy := r.parseUnitFile(tt.filename)
			if id != tt.wantID {
				t.Errorf("parseUnitFile() id = %q, want %q", id, tt.wantID)
			}
			if unitType != tt.wantType {
				t.Errorf("parseUnitFile() type = %q, want %q", unitType, tt.wantType)
			}
			if isLegacy != tt.wantLegacy {
				t.Errorf("parseUnitFile() isLegacy = %v, want %v", isLegacy, tt.wantLegacy)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"a1b2c3d4", true},
		{"z9y8x7w6", true},
		{"12345678", true},
		{"abcdefgh", true},
		{"", false},
		{"a1b2c3d", false},
		{"a1b2c3d4e5", false},
		{"A1B2C3D4", false},
		{"a1b2c3d!", false},
		{"a1b2c3d ", false},
		{"my-gdrive", false},
		{"my_gdrive", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := isValidID(tt.id); got != tt.want {
				t.Errorf("isValidID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestReconciler_RemoveOrphan(t *testing.T) {
	tmpDir := t.TempDir()
	serviceFile := filepath.Join(tmpDir, "rclone-mount-xyz12345.service")
	if err := os.WriteFile(serviceFile, []byte("[Unit]\nDescription=Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}
	m := NewManager()
	r := NewReconciler(g, m)

	orphan := OrphanedUnit{
		Name:     "rclone-mount-xyz12345.service",
		Type:     "mount",
		ID:       "xyz12345",
		IsLegacy: false,
		Path:     serviceFile,
	}

	err := r.RemoveOrphan(orphan)
	if err == nil {
		if _, err := os.Stat(serviceFile); !os.IsNotExist(err) {
			t.Error("RemoveOrphan() did not remove the file")
		}
	}
}

func TestReconciler_RemoveOrphan_WithTimer(t *testing.T) {
	tmpDir := t.TempDir()
	serviceFile := filepath.Join(tmpDir, "rclone-sync-abc12345.service")
	timerFile := filepath.Join(tmpDir, "rclone-sync-abc12345.timer")
	if err := os.WriteFile(serviceFile, []byte("[Unit]\nDescription=Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(timerFile, []byte("[Unit]\nDescription=Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}
	m := NewManager()
	r := NewReconciler(g, m)

	orphan := OrphanedUnit{
		Name:     "rclone-sync-abc12345.service",
		Type:     "sync",
		ID:       "abc12345",
		IsLegacy: false,
		Path:     serviceFile,
	}

	err := r.RemoveOrphan(orphan)
	if err == nil {
		if _, err := os.Stat(serviceFile); !os.IsNotExist(err) {
			t.Error("RemoveOrphan() did not remove service file")
		}
		if _, err := os.Stat(timerFile); !os.IsNotExist(err) {
			t.Error("RemoveOrphan() did not remove timer file")
		}
	}
}

func TestReconciler_Import_Mount(t *testing.T) {
	tmpDir := t.TempDir()
	serviceContent := `[Unit]
Description=Rclone mount: My Drive
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
ExecStart=/usr/bin/rclone mount gdrive:/ /mnt/gdrive --config=/home/user/.config/rclone/rclone.conf
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`
	serviceFile := filepath.Join(tmpDir, "rclone-mount-legacy1.service")
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}
	m := NewManager()
	r := NewReconciler(g, m)

	orphan := OrphanedUnit{
		Name:     "rclone-mount-legacy1.service",
		Type:     "mount",
		ID:       "legacy1",
		IsLegacy: true,
		Path:     serviceFile,
	}

	result, err := r.Import(orphan)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result == nil {
		t.Fatal("Import() returned nil")
	}
	if result.Mount == nil {
		t.Fatal("Import() Mount is nil")
	}
	if result.Mount.Name == "" {
		t.Error("Import() Mount.Name is empty")
	}
	if result.Mount.ID == "" {
		t.Error("Import() Mount.ID is empty")
	}
}

func TestReconciler_Import_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	serviceContent := `[Unit]
Description=Rclone sync: Backup Photos
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/bin/rclone sync gdrive:/Photos /home/user/Backup/Photos --config=/home/user/.config/rclone/rclone.conf

[Install]
WantedBy=default.target
`
	serviceFile := filepath.Join(tmpDir, "rclone-sync-legacy2.service")
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	timerContent := `[Unit]
Description=Timer for rclone sync: Backup Photos

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
`
	timerFile := filepath.Join(tmpDir, "rclone-sync-legacy2.timer")
	if err := os.WriteFile(timerFile, []byte(timerContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	g := &Generator{
		systemdDir: tmpDir,
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     tmpDir,
	}
	m := NewManager()
	r := NewReconciler(g, m)

	orphan := OrphanedUnit{
		Name:     "rclone-sync-legacy2.service",
		Type:     "sync",
		ID:       "legacy2",
		IsLegacy: true,
		Path:     serviceFile,
	}

	result, err := r.Import(orphan)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result == nil {
		t.Fatal("Import() returned nil")
	}
	if result.SyncJob == nil {
		t.Fatal("Import() SyncJob is nil")
	}
	if result.SyncJob.Name == "" {
		t.Error("Import() SyncJob.Name is empty")
	}
	if result.SyncJob.Schedule.Type != "timer" {
		t.Errorf("Import() SyncJob.Schedule.Type = %q, want 'timer'", result.SyncJob.Schedule.Type)
	}
}

func TestReconciler_Import_FileNotFound(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	orphan := OrphanedUnit{
		Name:     "rclone-mount-notfound.service",
		Type:     "mount",
		ID:       "notfound",
		IsLegacy: false,
		Path:     "/nonexistent/path/rclone-mount-notfound.service",
	}

	_, err := r.Import(orphan)
	if err == nil {
		t.Error("Import() should return error for nonexistent file")
	}
}

func TestParseMountUnit(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	content := `[Unit]
Description=Rclone mount: Test Mount
[Service]
ExecStart=/usr/bin/rclone mount remote:/path /mnt/point --config=/home/user/.config/rclone/rclone.conf
`
	orphan := OrphanedUnit{Type: "mount"}

	mount, err := r.parseMountUnit(content, orphan)
	if err != nil {
		t.Fatalf("parseMountUnit() error = %v", err)
	}
	if mount == nil {
		t.Fatal("parseMountUnit() returned nil")
	}
	if mount.Name == "" {
		t.Error("parseMountUnit() Name is empty")
	}
}

func TestParseSyncUnit(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	content := `[Unit]
Description=Rclone sync: Test Sync
[Service]
ExecStart=/usr/bin/rclone sync source:/path /dest/path --config=/home/user/.config/rclone/rclone.conf
`
	orphan := OrphanedUnit{
		Type: "sync",
		Path: "/tmp/rclone-sync-test.service",
	}

	job, err := r.parseSyncUnit(content, orphan)
	if err != nil {
		t.Fatalf("parseSyncUnit() error = %v", err)
	}
	if job == nil {
		t.Fatal("parseSyncUnit() returned nil")
	}
	if job.Name == "" {
		t.Error("parseSyncUnit() Name is empty")
	}
}

func TestExtractNameFromDescription(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		unitType string
		want     string
	}{
		{
			name: "mount description",
			content: `[Unit]
Description=Rclone mount: My Google Drive
`,
			unitType: "mount",
			want:     "My Google Drive",
		},
		{
			name: "sync description",
			content: `[Unit]
Description=Rclone sync: Backup Photos
`,
			unitType: "sync",
			want:     "Backup Photos",
		},
		{
			name: "no description",
			content: `[Unit]
After=network.target
`,
			unitType: "mount",
			want:     "imported-mount",
		},
		{
			name: "multiline with description",
			content: `[Unit]
After=network.target
Description=Rclone mount: Test Drive
Wants=network.target
`,
			unitType: "mount",
			want:     "Test Drive",
		},
		{
			name: "lowercase description",
			content: `[Unit]
Description=rclone mount: lowercase
`,
			unitType: "mount",
			want:     "lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNameFromDescription(tt.content, tt.unitType)
			if got != tt.want {
				t.Errorf("extractNameFromDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractExecStart(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantContains string
	}{
		{
			name: "simple exec",
			content: `[Service]
ExecStart=/usr/bin/rclone mount remote:/ /mnt
`,
			wantContains: "/usr/bin/rclone mount remote:/ /mnt",
		},
		{
			name: "multiline exec",
			content: `[Service]
ExecStart=/usr/bin/rclone mount remote:/ /mnt \
    --config=/home/user/.config/rclone/rclone.conf \
    --vfs-cache-mode=full
`,
			wantContains: "/usr/bin/rclone mount remote:/ /mnt",
		},
		{
			name: "no exec",
			content: `[Service]
Type=notify
`,
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExecStart(tt.content)
			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("extractExecStart() = %q, want to contain %q", got, tt.wantContains)
			}
			if tt.wantContains == "" && got != "" {
				t.Errorf("extractExecStart() = %q, want empty", got)
			}
		})
	}
}

func TestParseRcloneMountCommand(t *testing.T) {
	tests := []struct {
		name      string
		execStart string
		wantLen   int
	}{
		{
			name:      "valid mount command",
			execStart: "/usr/bin/rclone mount gdrive:/ /mnt/gdrive",
			wantLen:   2,
		},
		{
			name:      "mount with options",
			execStart: "/usr/bin/rclone mount remote:/path /mnt/point --vfs-cache-mode=full",
			wantLen:   2,
		},
		{
			name:      "invalid command",
			execStart: "/usr/bin/rclone sync source dest",
			wantLen:   0,
		},
		{
			name:      "empty command",
			execStart: "",
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRcloneMountCommand(tt.execStart)
			if len(got) != tt.wantLen {
				t.Errorf("parseRcloneMountCommand() returned %d elements, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParseRcloneSyncCommand(t *testing.T) {
	tests := []struct {
		name      string
		execStart string
		wantLen   int
	}{
		{
			name:      "sync command",
			execStart: "/usr/bin/rclone sync source:/path /dest/path",
			wantLen:   3,
		},
		{
			name:      "copy command",
			execStart: "/usr/bin/rclone copy source:/path /dest/path",
			wantLen:   3,
		},
		{
			name:      "move command",
			execStart: "/usr/bin/rclone move source:/path /dest/path",
			wantLen:   3,
		},
		{
			name:      "invalid command",
			execStart: "/usr/bin/rclone mount source dest",
			wantLen:   0,
		},
		{
			name:      "empty command",
			execStart: "",
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRcloneSyncCommand(tt.execStart)
			if len(got) != tt.wantLen {
				t.Errorf("parseRcloneSyncCommand() returned %d elements, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		name       string
		remotePath string
		wantRemote string
		wantPath   string
	}{
		{
			name:       "remote with path",
			remotePath: "gdrive:/Photos/2024",
			wantRemote: "gdrive:",
			wantPath:   "/Photos/2024",
		},
		{
			name:       "remote with root path",
			remotePath: "gdrive:/",
			wantRemote: "gdrive:",
			wantPath:   "/",
		},
		{
			name:       "remote without path",
			remotePath: "gdrive:",
			wantRemote: "gdrive:",
			wantPath:   "",
		},
		{
			name:       "no colon",
			remotePath: "localpath",
			wantRemote: "localpath",
			wantPath:   "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, path := parseRemotePath(tt.remotePath)
			if remote != tt.wantRemote {
				t.Errorf("parseRemotePath() remote = %q, want %q", remote, tt.wantRemote)
			}
			if path != tt.wantPath {
				t.Errorf("parseRemotePath() path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

func TestReconciler_ParseTimerSchedule(t *testing.T) {
	g := &Generator{
		systemdDir: t.TempDir(),
		rclonePath: "/usr/bin/rclone",
		configPath: "/home/user/.config/rclone/rclone.conf",
		logDir:     t.TempDir(),
	}
	m := NewManager()
	r := NewReconciler(g, m)

	tests := []struct {
		name        string
		content     string
		wantType    string
		wantCal     string
		wantPersist bool
	}{
		{
			name: "daily timer",
			content: `[Timer]
OnCalendar=daily
`,
			wantType:    "timer",
			wantCal:     "daily",
			wantPersist: false,
		},
		{
			name: "onboot timer",
			content: `[Timer]
OnBootSec=5min
`,
			wantType:    "onboot",
			wantCal:     "",
			wantPersist: false,
		},
		{
			name: "persistent timer",
			content: `[Timer]
OnCalendar=daily
Persistent=true
`,
			wantType:    "timer",
			wantCal:     "daily",
			wantPersist: true,
		},
		{
			name: "timer with delay",
			content: `[Timer]
OnCalendar=*-*-* 02:00:00
RandomizedDelaySec=10m
OnUnitActiveSec=1h
`,
			wantType:    "timer",
			wantCal:     "*-*-* 02:00:00",
			wantPersist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := r.parseTimerSchedule(tt.content)
			if config.Type != tt.wantType {
				t.Errorf("parseTimerSchedule() Type = %q, want %q", config.Type, tt.wantType)
			}
			if config.OnCalendar != tt.wantCal {
				t.Errorf("parseTimerSchedule() OnCalendar = %q, want %q", config.OnCalendar, tt.wantCal)
			}
			if config.Persistent != tt.wantPersist {
				t.Errorf("parseTimerSchedule() Persistent = %v, want %v", config.Persistent, tt.wantPersist)
			}
		})
	}
}

func TestGenerateNewID(t *testing.T) {
	id1 := generateNewID()
	id2 := generateNewID()

	if len(id1) != 8 {
		t.Errorf("generateNewID() length = %d, want 8", len(id1))
	}
	if len(id2) != 8 {
		t.Errorf("generateNewID() length = %d, want 8", len(id2))
	}
	if id1 == id2 {
		t.Error("generateNewID() generated identical IDs")
	}
}

func TestOrphanedUnit_Struct(t *testing.T) {
	orphan := OrphanedUnit{
		Name:     "rclone-mount-test.service",
		Type:     "mount",
		ID:       "test",
		IsLegacy: true,
		Path:     "/path/to/file",
		Imported: false,
	}

	if orphan.Name != "rclone-mount-test.service" {
		t.Errorf("OrphanedUnit.Name = %q, want %q", orphan.Name, "rclone-mount-test.service")
	}
	if orphan.Type != "mount" {
		t.Errorf("OrphanedUnit.Type = %q, want %q", orphan.Type, "mount")
	}
	if !orphan.IsLegacy {
		t.Error("OrphanedUnit.IsLegacy should be true")
	}
}

func TestImportedConfig_Struct(t *testing.T) {
	config := ImportedConfig{
		Mount: &models.MountConfig{
			ID:   "test1234",
			Name: "Test Mount",
		},
		Unit: OrphanedUnit{
			Name: "rclone-mount-test.service",
			Type: "mount",
		},
	}

	if config.Mount == nil {
		t.Error("ImportedConfig.Mount should not be nil")
	}
	if config.Mount.Name != "Test Mount" {
		t.Errorf("ImportedConfig.Mount.Name = %q, want %q", config.Mount.Name, "Test Mount")
	}
}

func TestReconciliationResult_Struct(t *testing.T) {
	result := ReconciliationResult{
		OrphanedUnits: []OrphanedUnit{
			{Name: "orphan1.service", Type: "mount"},
			{Name: "orphan2.service", Type: "sync"},
		},
		Errors: []error{},
	}

	if len(result.OrphanedUnits) != 2 {
		t.Errorf("ReconciliationResult.OrphanedUnits length = %d, want 2", len(result.OrphanedUnits))
	}
}
