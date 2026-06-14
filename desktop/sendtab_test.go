package main

import (
	"testing"
)

func TestParseSendTabArgs(t *testing.T) {
	cases := []struct {
		name          string
		args          string
		wantTab       string
		wantMessage   string
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name:        "simple unquoted",
			args:        "dev check this bug",
			wantTab:     "dev",
			wantMessage: "check this bug",
		},
		{
			name:        "quoted multi-word title",
			args:        `"My Project" hello there`,
			wantTab:     "My Project",
			wantMessage: "hello there",
		},
		{
			name:        "single word quoted title",
			args:        `"dev" check this`,
			wantTab:     "dev",
			wantMessage: "check this",
		},
		{
			name:          "missing args",
			args:          "",
			wantErr:       true,
			wantErrPrefix: "usage:",
		},
		{
			name:          "only title no message",
			args:          "dev",
			wantErr:       true,
			wantErrPrefix: "usage:",
		},
		{
			name:          "unclosed quote",
			args:          `"dev check this`,
			wantErr:       true,
			wantErrPrefix: "unclosed quote",
		},
		{
			name:          "empty message",
			args:          "dev   ",
			wantErr:       true,
			wantErrPrefix: "usage:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTab, gotMsg, err := parseSendTabArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if tc.wantErrPrefix != "" && len(err.Error()) < len(tc.wantErrPrefix) {
					t.Fatalf("error %q does not start with %q", err.Error(), tc.wantErrPrefix)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotTab != tc.wantTab {
				t.Errorf("tab = %q, want %q", gotTab, tc.wantTab)
			}
			if gotMsg != tc.wantMessage {
				t.Errorf("message = %q, want %q", gotMsg, tc.wantMessage)
			}
		})
	}
}
