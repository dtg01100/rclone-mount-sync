package rclone

import (
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
