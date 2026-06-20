// dkre_api provides a local HTTP API (127.0.0.1 only) that lets external
// processes send messages to open desktop tabs and query tab status.
//
// Endpoints:
//
//	GET  /dkre/api/tabs                 — list all open tabs with status
//	POST /dkre/api/tabs/{tabName}/send  — send a message to a tab by exact title
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	dkreAPIDefaultPort = "17532"
	dkreAPIEnvPort     = "REASONIX_DESKTOP_API_PORT"
)

// dkreAPI is a local HTTP API that lets external processes send messages to
// open desktop tabs and query tab status. It listens on 127.0.0.1 only.
type dkreAPI struct {
	app *App
	srv *http.Server
}

func newDkreAPI(app *App) *dkreAPI {
	return &dkreAPI{app: app}
}

// start begins listening on 127.0.0.1:<port> and shuts down when ctx is cancelled.
func (api *dkreAPI) start(ctx context.Context) {
	port := os.Getenv(dkreAPIEnvPort)
	if port == "" {
		port = dkreAPIDefaultPort
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /dkre/api/tabs", api.handleListTabs)
	mux.HandleFunc("POST /dkre/api/tabs/{tabName}/send", api.handleSendToTab)

	addr := net.JoinHostPort("127.0.0.1", port)
	api.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		slog.Info("dkre API server listening", "addr", "http://"+addr+"/dkre/api")
		if err := api.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("dkre API server error", "err", err)
		}
	}()

	// Shut down cleanly when the app context is cancelled.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = api.srv.Shutdown(shutdownCtx)
	}()
}

// TabStatusView is the JSON shape returned by GET /dkre/api/tabs.
type TabStatusView struct {
	Name            string `json:"name"`
	Running         bool   `json:"running"`
	PendingPrompt   bool   `json:"pendingPrompt"`
	BackgroundJobs  int    `json:"backgroundJobs"`
	CancelRequested bool   `json:"cancelRequested"`
	Cancellable     bool   `json:"cancellable"`
	LoopActive      bool   `json:"loopActive"`
	Ready           bool   `json:"ready"`
}

func (api *dkreAPI) handleListTabs(w http.ResponseWriter, r *http.Request) {
	tabs := api.app.ListTabs()
	out := make([]TabStatusView, 0, len(tabs))
	for _, t := range tabs {
		out = append(out, TabStatusView{
			Name:            t.TopicTitle,
			Running:         t.Running,
			PendingPrompt:   t.PendingPrompt,
			BackgroundJobs:  t.BackgroundJobs,
			CancelRequested: t.CancelRequested,
			Cancellable:     t.Cancellable,
			LoopActive:      t.LoopActive,
			Ready:           t.Ready,
		})
	}
	writeDkreJSON(w, http.StatusOK, out)
}

// sendMessageRequest is the JSON body expected by POST /dkre/api/tabs/{tabName}/send.
type sendMessageRequest struct {
	Message string `json:"message"`
}

func (api *dkreAPI) handleSendToTab(w http.ResponseWriter, r *http.Request) {
	tabName := r.PathValue("tabName")
	if strings.TrimSpace(tabName) == "" {
		writeDkreJSON(w, http.StatusBadRequest, map[string]string{"error": "tabName is required"})
		return
	}

	var body sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeDkreJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		writeDkreJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	// Use "" as sourceTabID; sourceTabTitle returns "dkre-api" for empty IDs.
	if err := sendToTabByTitle(api.app, "", tabName, body.Message); err != nil {
		status := http.StatusNotFound
		if strings.Contains(err.Error(), "currently busy") {
			status = http.StatusConflict // 409 — retry later
		}
		slog.Warn("dkre API: send failed", "tab", tabName, "err", err)
		writeDkreJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("dkre API: message sent", "tab", tabName, "len", len(body.Message))
	writeDkreJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func writeDkreJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("dkre API: failed to encode JSON response", "err", err)
	}
}

// Ensure dkreAPI implements a fmt.Stringer so the startup log line is neat.
func (api *dkreAPI) String() string {
	if api.srv == nil {
		return "dkre API (not started)"
	}
	return fmt.Sprintf("dkre API (http://127.0.0.1%s)", api.srv.Addr)
}

// stop shuts down the HTTP server immediately.
func (api *dkreAPI) stop() {
	if api.srv != nil {
		_ = api.srv.Close()
	}
}
