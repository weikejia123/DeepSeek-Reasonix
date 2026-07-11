package main

import (
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/event"
)

func TestWireEventTabPreservesSharedRetryingFields(t *testing.T) {
	w := toWireTab(event.Event{Kind: event.Retrying, RetryAttempt: 3, RetryMax: 10}, "tab-1")
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"diff":"@@ -27 +27 @@\n-old\n+new\n"`) || !strings.Contains(s, `"added":1`) || !strings.Contains(s, `"removed":1`) {
		t.Fatalf("tool file diff was not serialized: %s", s)
	}
}

func TestToWireToolResult(t *testing.T) {
	e := event.Event{Kind: event.ToolResult, Tool: event.Tool{ID: "1", Output: "ok", Truncated: true, DurationMs: 522}}
	w := toWire(e)
	if w.Tool == nil || w.Tool.Output != "ok" || !w.Tool.Truncated || w.Tool.DurationMs != 522 {
		t.Errorf("tool result = %+v", w.Tool)
	}
}

func TestToWireToolProgress(t *testing.T) {
	e := event.Event{Kind: event.ToolProgress, Tool: event.Tool{ID: "1", Output: "chunk"}}
	w := toWire(e)
	if w.Kind != "tool_progress" || w.Tool == nil || w.Tool.Output != "chunk" {
		t.Errorf("tool progress = kind:%q tool:%+v", w.Kind, w.Tool)
	}
}

func TestToWireUsage(t *testing.T) {
	e := event.Event{
		Kind:        event.Usage,
		Usage:       &provider.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, CacheHitTokens: 80, CacheMissTokens: 20},
		UsageSource: event.UsageSourceSubagent,
		SessionHit:  800,
		SessionMiss: 200,
	}
	w := toWire(e)
	if w.Usage == nil || w.Usage.PromptTokens != 100 || w.Usage.TotalTokens != 150 {
		t.Errorf("usage = %+v", w.Usage)
	}
	if w.Usage.Source != event.UsageSourceSubagent {
		t.Errorf("usage source = %q, want subagent", w.Usage.Source)
	}
	if w.Usage.SessionCacheHitTokens != 800 || w.Usage.SessionCacheMissTokens != 200 {
		t.Errorf("session cache = hit:%d miss:%d", w.Usage.SessionCacheHitTokens, w.Usage.SessionCacheMissTokens)
	}
}

func TestToWireUsageWithPricing(t *testing.T) {
	e := event.Event{
		Kind:    event.Usage,
		Usage:   &provider.Usage{CacheHitTokens: 1_000_000, CacheMissTokens: 0, CompletionTokens: 0},
		Pricing: &provider.Pricing{CacheHit: 1.0, Input: 2.0, Output: 10.0},
	}
	w := toWire(e)
	if w.Usage == nil || w.Usage.Cost != 1.0 || w.Usage.CostUSD != 1.0 {
		t.Errorf("cost = %+v, want cost and compat costUsd of 1.0", w.Usage)
	}
	if w.Usage.Currency != "¥" {
		t.Errorf("currency = %q, want ¥", w.Usage.Currency)
	}
}

func TestToWireApprovalRequest(t *testing.T) {
	e := event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{ID: "42", Tool: "bash", Subject: "rm"}}
	w := toWire(e)
	if w.Approval == nil || w.Approval.ID != "42" || w.Approval.Tool != "bash" {
		t.Errorf("approval = %+v", w.Approval)
	}
}

func TestToWireAskRequest(t *testing.T) {
	e := event.Event{Kind: event.AskRequest, Ask: event.Ask{
		ID:        "ask-1",
		Questions: []event.AskQuestion{{ID: "q1", Header: "Pick", Prompt: "Choose one", Options: []event.AskOption{{Label: "A"}, {Label: "B"}}, Multi: false}},
	}}
	w := toWire(e)
	if w.Ask == nil || w.Ask.ID != "ask-1" {
		t.Errorf("ask = %+v", w.Ask)
	}
	if len(w.Ask.Questions) != 1 || len(w.Ask.Questions[0].Options) != 2 {
		t.Errorf("questions/options = %+v", w.Ask.Questions)
	}
}

func TestToWireTurnDoneWithError(t *testing.T) {
	e := event.Event{Kind: event.TurnDone, Err: errors.New("boom")}
	w := toWire(e)
	if w.Kind != "turn_done" || w.Err != "boom" {
		t.Errorf("turn_done error = %+v", w)
	}
}

func TestToWireTurnDoneNoError(t *testing.T) {
	e := event.Event{Kind: event.TurnDone}
	w := toWire(e)
	if w.Err != "" {
		t.Errorf("turn_done no-error should have empty err, got %q", w.Err)
	}
}

// --- kindNames completeness ---

func TestToWireSteer(t *testing.T) {
	e := event.Event{Kind: event.Steer, Text: "please use smaller diffs"}
	w := toWire(e)
	if w.Kind != "steer" || w.Text != "please use smaller diffs" {
		t.Errorf("steer wire = %+v", w)
	}
}

func TestKindNamesComplete(t *testing.T) {
	// UserMessage is the last Kind; every value through it must have a wire name,
	// or toWire emits kind:"" and the frontend reducer falls through to undefined.
	for k := event.Kind(0); k <= event.UserMessage; k++ {
		if kindNames[k] == "" {
			t.Errorf("kind %d has no wire name — toWire would emit kind:\"\"", k)
=======
	for _, want := range []string{`"kind":"retrying"`, `"retryAttempt":3`, `"retryMax":10`, `"tabId":"tab-1"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("tab retrying JSON = %s, want it to contain %s", s, want)
		}
	}
}
