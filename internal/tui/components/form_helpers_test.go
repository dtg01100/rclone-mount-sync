package components

import (
	"errors"
	"testing"
)

type mockRemoteLister struct {
	dirs []string
	err  error
}

func (m *mockRemoteLister) ListRootDirectories(remoteName string) ([]string, error) {
	return m.dirs, m.err
}

func TestValidateBufferSize(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid megabytes",
			value:   "16M",
			wantErr: false,
		},
		{
			name:    "valid gigabytes",
			value:   "1G",
			wantErr: false,
		},
		{
			name:    "valid kilobytes",
			value:   "512K",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase m",
			value:   "16m",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase g",
			value:   "1g",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase k",
			value:   "512k",
			wantErr: false,
		},
		{
			name:    "valid large value",
			value:   "2048M",
			wantErr: false,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
			errType: errors.New("buffer size cannot be empty"),
		},
		{
			name:    "number only without unit",
			value:   "16",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			value:   "16X",
			wantErr: true,
		},
		{
			name:    "zero value",
			value:   "0M",
			wantErr: true,
		},
		{
			name:    "negative value",
			value:   "-1M",
			wantErr: true,
		},
		{
			name:    "decimal value",
			value:   "1.5M",
			wantErr: true,
		},
		{
			name:    "unit first",
			value:   "M16",
			wantErr: true,
		},
		{
			name:    "letters in number",
			value:   "16MM",
			wantErr: true,
		},
		{
			name:    "just letters",
			value:   "ABC",
			wantErr: true,
		},
		{
			name:    "special characters",
			value:   "16@M",
			wantErr: true,
		},
		{
			name:    "space in value",
			value:   "16 M",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBufferSize(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBufferSize(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && err != nil {
				if err.Error() != tt.errType.Error() {
					t.Errorf("ValidateBufferSize(%q) error = %v, want error %v", tt.value, err, tt.errType)
				}
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid hours",
			value:   "24h",
			wantErr: false,
		},
		{
			name:    "valid minutes",
			value:   "30m",
			wantErr: false,
		},
		{
			name:    "valid seconds",
			value:   "5s",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase h",
			value:   "24h",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase m",
			value:   "30m",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase s",
			value:   "5s",
			wantErr: false,
		},
		{
			name:    "valid large hours",
			value:   "168h",
			wantErr: false,
		},
		{
			name:    "valid large minutes",
			value:   "1440m",
			wantErr: false,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "number only without unit",
			value:   "30",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			value:   "30x",
			wantErr: true,
		},
		{
			name:    "zero value",
			value:   "0h",
			wantErr: true,
		},
		{
			name:    "negative value",
			value:   "-1h",
			wantErr: true,
		},
		{
			name:    "decimal value",
			value:   "1.5h",
			wantErr: true,
		},
		{
			name:    "unit first",
			value:   "h24",
			wantErr: true,
		},
		{
			name:    "multiple letters in unit",
			value:   "30hr",
			wantErr: true,
		},
		{
			name:    "just letters",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "special characters",
			value:   "30@h",
			wantErr: true,
		},
		{
			name:    "space in value",
			value:   "30 h",
			wantErr: true,
		},
		{
			name:    "unit uppercase - case insensitive regex",
			value:   "30M",
			wantErr: false,
		},
		{
			name:    "unit uppercase H - case insensitive regex",
			value:   "30H",
			wantErr: false,
		},
		{
			name:    "unit uppercase S - case insensitive regex",
			value:   "30S",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDuration(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDuration(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUmask(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid 3 digits",
			value:   "002",
			wantErr: false,
		},
		{
			name:    "valid 3 digits with leading zero",
			value:   "022",
			wantErr: false,
		},
		{
			name:    "valid 4 digits",
			value:   "0002",
			wantErr: false,
		},
		{
			name:    "valid all zeros",
			value:   "000",
			wantErr: false,
		},
		{
			name:    "valid max value",
			value:   "0777",
			wantErr: false,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "single digit",
			value:   "2",
			wantErr: true,
		},
		{
			name:    "two digits",
			value:   "02",
			wantErr: true,
		},
		{
			name:    "five digits",
			value:   "00002",
			wantErr: true,
		},
		{
			name:    "contains 8",
			value:   "082",
			wantErr: true,
		},
		{
			name:    "contains 9",
			value:   "092",
			wantErr: true,
		},
		{
			name:    "contains letters",
			value:   "00a",
			wantErr: true,
		},
		{
			name:    "contains special characters",
			value:   "00-",
			wantErr: true,
		},
		{
			name:    "space in value",
			value:   "00 2",
			wantErr: true,
		},
		{
			name:    "starts with 8",
			value:   "800",
			wantErr: true,
		},
		{
			name:    "negative number",
			value:   "-022",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUmask(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUmask(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBandwidthLimit(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid empty string",
			value:   "",
			wantErr: false,
		},
		{
			name:    "valid megabytes",
			value:   "10M",
			wantErr: false,
		},
		{
			name:    "valid gigabytes",
			value:   "1G",
			wantErr: false,
		},
		{
			name:    "valid kilobytes",
			value:   "512K",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase m",
			value:   "10m",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase g",
			value:   "1g",
			wantErr: false,
		},
		{
			name:    "valid case insensitive lowercase k",
			value:   "512k",
			wantErr: false,
		},
		{
			name:    "valid large value",
			value:   "100M",
			wantErr: false,
		},
		{
			name:    "number only without unit",
			value:   "10",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			value:   "10X",
			wantErr: true,
		},
		{
			name:    "zero value",
			value:   "0M",
			wantErr: true,
		},
		{
			name:    "negative value",
			value:   "-1M",
			wantErr: true,
		},
		{
			name:    "decimal value",
			value:   "1.5M",
			wantErr: true,
		},
		{
			name:    "unit first",
			value:   "M10",
			wantErr: true,
		},
		{
			name:    "letters in number",
			value:   "10MM",
			wantErr: true,
		},
		{
			name:    "just letters",
			value:   "ABC",
			wantErr: true,
		},
		{
			name:    "special characters",
			value:   "10@M",
			wantErr: true,
		},
		{
			name:    "space in value",
			value:   "10 M",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			value:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBandwidthLimit(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBandwidthLimit(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestGetRemotePathSuggestions(t *testing.T) {
	tests := []struct {
		name            string
		rcloneClient    interface{}
		remoteName      string
		staticFallbacks []string
		want            []string
	}{
		{
			name:            "nil client with empty fallbacks",
			rcloneClient:    nil,
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{},
		},
		{
			name:            "nil client with fallbacks",
			rcloneClient:    nil,
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data", "/media/usb"},
			want:            []string{"/mnt/data", "/media/usb"},
		},
		{
			name: "client returns directories",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"/dir1", "/dir2"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{"/dir1", "/dir2"},
		},
		{
			name: "client returns directories with fallbacks",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"/dir1", "/dir2"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data", "/media/usb"},
			want:            []string{"/dir1", "/dir2", "/mnt/data", "/media/usb"},
		},
		{
			name: "client returns error - fall back to static",
			rcloneClient: &mockRemoteLister{
				dirs: []string{},
				err:  errors.New("connection error"),
			},
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data"},
			want:            []string{"/mnt/data"},
		},
		{
			name:            "empty fallback list",
			rcloneClient:    &mockRemoteLister{dirs: []string{"/dir1"}},
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{"/dir1"},
		},
		{
			name:            "nil client nil fallbacks",
			rcloneClient:    nil,
			remoteName:      "remote",
			staticFallbacks: nil,
			want:            []string{},
		},
		{
			name: "deduplicates client and fallback",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"/shared"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{"/shared"},
			want:            []string{"/shared"},
		},
		{
			name: "trims trailing slash from client dirs",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"/dir1/", "/dir2/"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{"/dir1", "/dir2"},
		},
		{
			name:            "trims trailing slash from fallbacks",
			rcloneClient:    nil,
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data/", "/media/usb/"},
			want:            []string{"/mnt/data/", "/media/usb/"},
		},
		{
			name: "filters empty client dirs",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"", "/dir1", "", "/dir2"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{"/dir1", "/dir2"},
		},
		{
			name:            "filters empty fallbacks",
			rcloneClient:    nil,
			remoteName:      "remote",
			staticFallbacks: []string{"", "/mnt/data", "", "/media/usb"},
			want:            []string{"/mnt/data", "/media/usb"},
		},
		{
			name:            "non-RemoteLister client type",
			rcloneClient:    "not a remote lister",
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data"},
			want:            []string{"/mnt/data"},
		},
		{
			name:            "non-RemoteLister client nil",
			rcloneClient:    12345,
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data"},
			want:            []string{"/mnt/data"},
		},
		{
			name:            "empty client dirs list",
			rcloneClient:    &mockRemoteLister{dirs: []string{}},
			remoteName:      "remote",
			staticFallbacks: []string{"/mnt/data"},
			want:            []string{"/mnt/data"},
		},
		{
			name: "client with slash prefixes in dirs",
			rcloneClient: &mockRemoteLister{
				dirs: []string{"/path1", "path2"},
			},
			remoteName:      "remote",
			staticFallbacks: []string{},
			want:            []string{"/path1", "path2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRemotePathSuggestions(tt.rcloneClient, tt.remoteName, tt.staticFallbacks)
			if len(got) != len(tt.want) {
				t.Errorf("GetRemotePathSuggestions() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetRemotePathSuggestions()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
