package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"reasonix/internal/command"
	"reasonix/internal/event"
)

const (
	sendtabName        = "sendtab"
	sendtabDescription = "Send a message to a specific open tab by its exact title."
	sendtabArgHint     = "<tab-title> <message>"
)

// messageFromTabPrefix is the display prefix used when a message is injected
// into a tab from another tab. The frontend detects this prefix to render the
// message with a distinctive (blue) timeline marker, similar to A-Loop.
const messageFromTabPrefix = "MessageFromTab"

// sendTabTool lets the model send a message to a specific open tab by its exact
// title. Desktop-only; requires the tab to already be open.
type sendTabTool struct {
	app         *App
	sourceTabID string
}

// newSendTabTool creates a sendTabTool bound to the desktop app instance and
// the tab whose controller owns this tool.
func newSendTabTool(app *App, sourceTabID string) *sendTabTool {
	return &sendTabTool{app: app, sourceTabID: sourceTabID}
}

// Name implements tool.Tool.
func (t *sendTabTool) Name() string { return sendtabName }

// ReadOnly implements tool.Tool. It is false because the tool sends a message.
func (t *sendTabTool) ReadOnly() bool { return false }

// Description implements tool.Tool.
func (t *sendTabTool) Description() string {
	return sendtabDescription + " Desktop only. The tab must already be open."
}

// Schema implements tool.Tool.
func (t *sendTabTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"tabName": {
				"type": "string",
				"description": "The exact title of the target tab (case-sensitive)."
			},
			"message": {
				"type": "string",
				"description": "The message text to send to the tab."
			}
		},
		"required": ["tabName", "message"]
	}`)
}

// Execute implements tool.Tool. It looks up the tab by exact title match and
// sends the message via SubmitToTab.
func (t *sendTabTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var p struct {
		TabName string `json:"tabName"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if err := sendToTabByTitle(t.app, t.sourceTabID, strings.TrimSpace(p.TabName), strings.TrimSpace(p.Message)); err != nil {
		return "", err
	}
	return fmt.Sprintf("message sent to tab %q", p.TabName), nil
}

// sendToTabByTitle finds an open tab by exact title and submits a message to it.
// The message is shown in the target tab's chat history with a
// "MessageFromTab[<source>]:" prefix and a distinctive timeline marker.
func sendToTabByTitle(app *App, sourceTabID, targetTabName, message string) error {
	if targetTabName == "" {
		return errors.New("tabName is required")
	}
	if message == "" {
		return errors.New("message is required")
	}

	sourceTitle := sourceTabTitle(app, sourceTabID)
	displayText := fmt.Sprintf("%s[%s]: %s", messageFromTabPrefix, sourceTitle, message)

	tabs := app.ListTabs()
	for i := range tabs {
		if tabs[i].TopicTitle != targetTabName {
			continue
		}
		tab := app.tabByID(tabs[i].ID)
		if tab == nil {
			return fmt.Errorf("no open tab with title %q", targetTabName)
		}
		if tab.Ctrl == nil {
			return fmt.Errorf("tab %q is not ready yet", targetTabName)
		}
		if tab.sink == nil {
			return fmt.Errorf("tab %q has no event sink", targetTabName)
		}

		// Emit UserMessage so the frontend renders a user bubble immediately,
		// then submit the raw message as a turn so the model responds.
		tab.sink.Emit(event.Event{Kind: event.UserMessage, Text: displayText})
		tab.Ctrl.SubmitDisplay(displayText, message)
		return nil
	}
	return fmt.Errorf("no open tab with title %q", targetTabName)
}

// sourceTabTitle returns the title of the source tab, or a fallback label if
// the tab cannot be found.
func sourceTabTitle(app *App, tabID string) string {
	if tab := app.tabByID(tabID); tab != nil && strings.TrimSpace(tab.TopicTitle) != "" {
		return tab.TopicTitle
	}
	return "unknown"
}

// newSendTabCommand returns a command.Command entry that surfaces /sendtab in
// the composer's slash menu. The actual execution is handled by the slash
// handler registered alongside it, so this command's body is only a fallback
// usage prompt.
func newSendTabCommand() command.Command {
	return command.Command{
		Name:        sendtabName,
		Description: sendtabDescription,
		ArgHint:     sendtabArgHint,
		Body:        "Usage: /sendtab <tab-title> <message>",
		Source:      "desktop",
	}
}

// parseSendTabArgs parses the argument text for /sendtab. It supports:
//   - /sendtab title message here
//   - /sendtab "multi word title" message here
//
// It returns the tab title and the message, or an error if the args are
// malformed.
func parseSendTabArgs(args string) (tabName, message string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", errors.New("usage: /sendtab <tab-title> <message>")
	}

	// Quoted title
	if strings.HasPrefix(args, `"`) {
		end := strings.Index(args[1:], `"`)
		if end == -1 {
			return "", "", errors.New("unclosed quote in tab title")
		}
		tabName = strings.TrimSpace(args[1 : end+1])
		message = strings.TrimSpace(args[end+2:])
	} else {
		// Unquoted: first token is the title, rest is the message.
		fields := strings.Fields(args)
		if len(fields) < 2 {
			return "", "", errors.New("usage: /sendtab <tab-title> <message>")
		}
		tabName = fields[0]
		message = strings.TrimSpace(args[len(fields[0]):])
	}

	if tabName == "" {
		return "", "", errors.New("tabName is required")
	}
	if message == "" {
		return "", "", errors.New("message is required")
	}
	return tabName, message, nil
}

// newSendTabSlashHandler returns a slash handler that immediately sends a
// message to the named open tab.
func newSendTabSlashHandler(app *App, sourceTabID string) func(args string) error {
	return func(args string) error {
		tabName, message, err := parseSendTabArgs(args)
		if err != nil {
			return err
		}
		if err := sendToTabByTitle(app, sourceTabID, tabName, message); err != nil {
			return err
		}
		app.noticeForTab("", fmt.Sprintf("sent message to tab %q", tabName))
		return nil
	}
}
