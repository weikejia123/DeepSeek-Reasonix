package main

import (
	"fmt"
	"testing"
)

func TestParseSendTabArgs(t *testing.T) {
	cases := []struct {
		name        string
		args        string
		wantTab     string
		wantMessage string
		wantErr     bool
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
			name:    "missing args",
			args:    "",
			wantErr: true,
		},
		{
			name:    "only title no message",
			args:    "dev",
			wantErr: true,
		},
		{
			name:    "unclosed quote",
			args:    `"dev check this`,
			wantErr: true,
		},
		{
			name:    "empty message",
			args:    "dev   ",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTab, gotMsg, err := parseSendTabArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
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

func TestMessageFromTabPrefix(t *testing.T) {
	if messageFromTabPrefix != "MessageFromTab" {
		t.Fatalf("messageFromTabPrefix = %q, want %q", messageFromTabPrefix, "MessageFromTab")
	}
}

func TestMessageFromTabDisplayFormat(t *testing.T) {
	// The format must be MessageFromTab[<title>]: <message> so the frontend can
	// parse the source tab title and apply the blue timeline marker.
	got := fmt.Sprintf("%s[%s]: %s", messageFromTabPrefix, "DERE-1.7.0", "这是一条测试记录，不用过度思考。")
	want := "MessageFromTab[DERE-1.7.0]: 这是一条测试记录，不用过度思考。"
	if got != want {
		t.Fatalf("display = %q, want %q", got, want)
	}
}

