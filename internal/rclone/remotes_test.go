package rclone

import (
	"context"
	"strings"
	"testing"
)

func TestValidateRemoteName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
		errSub  string
	}{
		// Valid names
		{"simple alpha", "gdrive", false, ""},
		{"digits", "drive123", false, ""},
		{"underscore", "my_remote", false, ""},
		{"dash", "my-remote", false, ""},
		{"dot", "remote.v2", false, ""},
		{"at sign", "user@host", false, ""},
		{"plus", "s3+bucket", false, ""},
		{"bang", "weird!", false, ""},
		{"mixed", "Gdrive_Photos-2024", false, ""},

		// Invalid: empty
		{"empty", "", true, "empty"},

		// Invalid: leading dash (looks like a flag)
		{"leading dash", "-bad", true, "must not start with -"},

		// Invalid: colon (would smuggle a different remote)
		{"contains colon", "name:other", true, "must not contain a colon"},

		// Invalid: shell metacharacters
		{"contains space", "my remote", true, "illegal character"},
		{"contains semicolon", "name;ls", true, "illegal character"},
		{"contains backtick", "name`id`", true, "illegal character"},
		{"contains dollar", "name$var", true, "illegal character"},
		{"contains slash", "name/path", true, "illegal character"},
		{"contains quote", "name\"x", true, "illegal character"},
		{"contains newline", "name\nfoo", true, "illegal character"},
		{"contains tab", "name\tfoo", true, "illegal character"},
		{"contains ampersand", "a&b", true, "illegal character"},
		{"contains pipe", "a|b", true, "illegal character"},
		{"contains parens", "(name)", true, "illegal character"},
		{"contains wildcard", "a*b", true, "illegal character"},
		{"contains bracket", "a[b", true, "illegal character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRemoteName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateRemoteName(%q) err = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if tc.wantErr && tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
				t.Errorf("ValidateRemoteName(%q) err = %q, want substring %q", tc.input, err, tc.errSub)
			}
		})
	}
}

func TestGetAllRemoteTypes(t *testing.T) {
	cases := []struct {
		name      string
		output    string
		wantTypes map[string]string
	}{
		{
			name: "multiple remotes happy path",
			output: `; --------------------
; gdrive
; --------------------
[gdrive]
type = drive
client_id =

; --------------------
; dropbox
; --------------------
[dropbox]
type = dropbox

[s3]
type = s3
`,
			wantTypes: map[string]string{
				"gdrive":  "drive",
				"dropbox": "dropbox",
				"s3":      "s3",
			},
		},
		{
			name: "no type line in section",
			output: `[broken]
client_id = xxx
`,
			wantTypes: map[string]string{},
		},
		{
			name: "empty type value is skipped",
			output: `[gdrive]
type =
[dropbox]
type = dropbox
`,
			wantTypes: map[string]string{
				"dropbox": "dropbox",
			},
		},
		{
			name: "section without name is skipped",
			output: `[]
type = drive
[real]
type = sftp
`,
			wantTypes: map[string]string{
				"real": "sftp",
			},
		},
		{
			name: "type with no space around equals",
			output: `[gdrive]
type=drive
[dropbox]
type=dropbox
`,
			wantTypes: map[string]string{
				"gdrive":  "drive",
				"dropbox": "dropbox",
			},
		},
		{
			name: "extra whitespace around equals",
			output: `[gdrive]
type    =    drive
[dropbox]
type = dropbox
`,
			wantTypes: map[string]string{
				"gdrive":  "drive",
				"dropbox": "dropbox",
			},
		},
		{
			name: "type not on its own line is ignored",
			output: `[gdrive]
mimetype = type=foo
type = drive
`,
			wantTypes: map[string]string{
				"gdrive": "drive",
			},
		},
		{
			name: "section header with trailing comment",
			output: `[gdrive] # primary
type = drive
[backup]
type = s3
`,
			wantTypes: map[string]string{
				"gdrive": "drive",
				"backup": "s3",
			},
		},
		{
			name:      "empty output",
			output:    "",
			wantTypes: map[string]string{},
		},
		{
			name:      "only comments",
			output:    "; just a comment\n",
			wantTypes: map[string]string{},
		},
		{
			// Map overwrite semantics mean the last type wins.
			// Real rclone configs only have one type per section,
			// but the parser must not panic on duplicates.
			name: "last type wins when duplicated",
			output: `[gdrive]
type = drive
type = bogus
`,
			wantTypes: map[string]string{
				"gdrive": "bogus",
			},
		},
		{
			name: "type line before any section is ignored",
			output: `type = drive
[real]
type = s3
`,
			wantTypes: map[string]string{
				"real": "s3",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Build a mock rclone that echoes the canned output for
			// `config show`. Use a per-test tmpdir so each subtest
			// has its own script.
			mockScript := "#!/bin/sh\n" +
				"if [ \"$1\" = \"config\" ] && [ \"$2\" = \"show\" ]; then\n" +
				"cat <<'__EOF__'\n" + tc.output + "__EOF__\n" +
				"fi\n"
			mockPath := createMockRclone(t, mockScript)
			c := NewClientWithPath(mockPath)

			got, err := c.GetAllRemoteTypes(context.Background())
			if err != nil {
				t.Fatalf("GetAllRemoteTypes() unexpected error: %v", err)
			}

			if len(got) != len(tc.wantTypes) {
				t.Errorf("GetAllRemoteTypes() returned %d types, want %d (got=%v want=%v)",
					len(got), len(tc.wantTypes), got, tc.wantTypes)
			}
			for name, want := range tc.wantTypes {
				if got[name] != want {
					t.Errorf("GetAllRemoteTypes()[%q] = %q, want %q", name, got[name], want)
				}
			}
		})
	}
}

func TestGetAllRemoteTypesCommandError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "config file not found" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.GetAllRemoteTypes(context.Background())
	if err == nil {
		t.Fatal("GetAllRemoteTypes() expected error from rclone, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get remote types") {
		t.Errorf("GetAllRemoteTypes() error = %q, want substring 'failed to get remote types'", err.Error())
	}
}

func TestGetAllRemoteTypesNilContext(t *testing.T) {
	// nil context should be replaced with Background; verify no panic.
	mockScript := `#!/bin/sh
[ "$1" = "config" ] && [ "$2" = "show" ] && {
	echo "[gdrive]"
	echo "type = drive"
}
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	got, err := c.GetAllRemoteTypes(nil) //nolint:staticcheck
	if err != nil {
		t.Fatalf("GetAllRemoteTypes(nil) unexpected error: %v", err)
	}
	if got["gdrive"] != "drive" {
		t.Errorf("GetAllRemoteTypes(nil)[gdrive] = %q, want %q", got["gdrive"], "drive")
	}
}

func TestGetAllRemoteTypesContextCancelled(t *testing.T) {
	// Use a context that's already cancelled; the call should
	// surface context.Canceled before doing any real work.
	mockScript := `#!/bin/sh
# Sleep so the timeout/cancel can fire.
sleep 5
[ "$1" = "config" ] && [ "$2" = "show" ] && echo "[gdrive]"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetAllRemoteTypes(ctx)
	if err == nil {
		t.Fatal("GetAllRemoteTypes() expected error from cancelled context, got nil")
	}
}
