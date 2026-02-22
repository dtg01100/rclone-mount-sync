package systemd

// MountServiceTemplate is the systemd service unit template for mounts.
const MountServiceTemplate = `[Unit]
Description=Rclone mount: {{.Name}}
Documentation=man:rclone(1)
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=30
StartLimitBurst=5

[Service]
Type=notify
ExecStartPre=/bin/mkdir -p {{.MountPoint}}
ExecStart={{.RclonePath}} mount \
    {{.Remote}}{{.RemotePath}} \
    {{.MountPoint}} \
    {{.MountOptions}}
ExecStop=/bin/fusermount -u {{.MountPoint}}
ExecStopPost=/bin/rmdir {{.MountPoint}}
Restart=on-failure
RestartSec=5s
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
NoNewPrivileges=true

[Install]
WantedBy=default.target
`

// SyncServiceTemplate is the systemd service unit template for sync jobs.
const SyncServiceTemplate = `[Unit]
Description=Rclone sync: {{.Name}}
Documentation=man:rclone(1)
After=network-online.target
Wants=network-online.target
{{if .RequireACPower}}ConditionACPower=true
{{end}}
[Service]
Type=oneshot
{{if .RequireUnmetered}}ExecCondition=/bin/sh -c 'test "$(dbus-send --system --print-reply=literal --dest=org.freedesktop.NetworkManager /org/freedesktop/NetworkManager org.freedesktop.DBus.Properties.Get string:org.freedesktop.NetworkManager string:Metered 2>/dev/null | grep -o "\"[0-9]*\"" | tr -d "\"")" != "4" || exit 0; exit 1'
{{end}}ExecStart={{.RclonePath}} {{.Direction}} \
    {{.Source}} \
    {{.Destination}} \
    {{.SyncOptions}}
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
MemoryMax=1G
CPUQuota=50%

[Install]
WantedBy=default.target
`

// SyncTimerTemplate is the systemd timer unit template for sync jobs.
const SyncTimerTemplate = `[Unit]
Description=Timer for rclone sync: {{.Name}}
Documentation=man:rclone(1)

[Timer]
{{.TimerDirectives}}

[Install]
WantedBy=timers.target
`

// MountUnitData contains data for mount service unit generation.
type MountUnitData struct {
	Name         string
	Remote       string
	RemotePath   string
	MountPoint   string
	ConfigPath   string
	MountOptions string
	LogLevel     string
	LogPath      string
	RclonePath   string
}

// SyncUnitData contains data for sync service unit generation.
type SyncUnitData struct {
	Name             string
	Source           string
	Destination      string
	Direction        string
	ConfigPath       string
	SyncOptions      string
	LogLevel         string
	LogPath          string
	RclonePath       string
	RequireACPower   bool
	RequireUnmetered bool
	ExecCondition    string
}

// TimerUnitData contains data for timer unit generation.
type TimerUnitData struct {
	Name            string
	TimerDirectives string
}
