package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// Patcher applies auto-sync and note changes. Implemented against the
// Kubernetes API in production and against the in-memory store in demo mode.
type Patcher interface {
	SetAutoSync(ctx context.Context, name string, enabled bool, note, who string) error
	SetNote(ctx context.Context, name, note string) error
}

// Server holds the dependencies of the HTTP handlers.
type Server struct {
	Store    *Store
	Hub      *Hub
	Patcher  Patcher
	Identity IdentityConfig
}

// HandleEvents streams the application list and subsequent changes as
// server-sent events: a "snapshot" event on connect, then "app" and
// "delete" events as things change.
func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)

		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.Hub.Subscribe()
	defer s.Hub.Unsubscribe(ch)

	fmt.Fprint(w, "retry: 3000\n\n")
	w.Write(formatSSE("snapshot", s.Store.Snapshot()))
	flusher.Flush()

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// HandleAutoSync pauses or resumes auto-sync for one application, recording
// who did it and an optional note.
func (s *Server) HandleAutoSync(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, ok := s.Store.Get(name); !ok {
		http.Error(w, "unknown application", http.StatusNotFound)

		return
	}
	var req struct {
		Enabled bool   `json:"enabled"`
		Note    string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}
	who := identityFromRequest(r, s.Identity)
	if err := s.Patcher.SetAutoSync(r.Context(), name, req.Enabled, req.Note, who); err != nil {
		slog.Error("failed to update auto-sync", "app", name, "enabled", req.Enabled, "error", err)
		http.Error(w, fmt.Sprintf("failed to update application: %v", err), http.StatusInternalServerError)

		return
	}
	slog.Info("auto-sync updated", "app", name, "enabled", req.Enabled, "by", who)
	w.WriteHeader(http.StatusNoContent)
}

// HandleNote sets or clears the pause note for one application.
func (s *Server) HandleNote(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, ok := s.Store.Get(name); !ok {
		http.Error(w, "unknown application", http.StatusNotFound)

		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}
	if err := s.Patcher.SetNote(r.Context(), name, req.Note); err != nil {
		slog.Error("failed to update note", "app", name, "error", err)
		http.Error(w, fmt.Sprintf("failed to update note: %v", err), http.StatusInternalServerError)

		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// K8sPatcher applies changes as JSON merge patches on Application resources.
type K8sPatcher struct {
	Client    dynamic.Interface
	Namespace string
}

func (p *K8sPatcher) patch(ctx context.Context, name string, patch map[string]any) error {
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = p.Client.Resource(applicationGVR).Namespace(p.Namespace).
		Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{FieldManager: "syncwatch"})

	return err
}

// SetAutoSync flips spec.syncPolicy.automated.enabled and updates the pause
// annotations in the same atomic patch.
func (p *K8sPatcher) SetAutoSync(ctx context.Context, name string, enabled bool, note, who string) error {
	var annotations map[string]any
	if enabled {
		// Resuming: clear the pause bookkeeping.
		annotations = map[string]any{annNote: nil, annPausedBy: nil, annPausedAt: nil}
	} else {
		// Pausing: a fresh pause always overwrites all three, so bookkeeping
		// from an earlier pause can never leak into this one.
		annotations = map[string]any{
			annNote:     nullable(note),
			annPausedBy: nullable(who),
			annPausedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}

	return p.patch(ctx, name, map[string]any{
		"metadata": map[string]any{"annotations": annotations},
		"spec": map[string]any{
			"syncPolicy": map[string]any{
				"automated": map[string]any{"enabled": enabled},
			},
		},
	})
}

// SetNote sets or clears the pause-note annotation.
func (p *K8sPatcher) SetNote(ctx context.Context, name, note string) error {
	return p.patch(ctx, name, map[string]any{
		"metadata": map[string]any{"annotations": map[string]any{annNote: nullable(note)}},
	})
}

// ClearPauseAnnotations removes all pause bookkeeping, used when auto-sync
// was re-enabled by a tool other than SyncWatch.
func (p *K8sPatcher) ClearPauseAnnotations(ctx context.Context, name string) error {
	return p.patch(ctx, name, map[string]any{
		"metadata": map[string]any{"annotations": map[string]any{
			annNote: nil, annPausedBy: nil, annPausedAt: nil,
		}},
	})
}

// StampExternalPause records when and by what auto-sync was paused, used
// when the pause was done by a tool other than SyncWatch.
func (p *K8sPatcher) StampExternalPause(ctx context.Context, name, who string) error {
	return p.patch(ctx, name, map[string]any{
		"metadata": map[string]any{"annotations": map[string]any{
			annNote:     nil, // an external pause is a fresh pause: no note yet
			annPausedBy: nullable(who),
			annPausedAt: time.Now().UTC().Format(time.RFC3339),
		}},
	})
}

// nullable returns the value for a merge-patch annotation: the string itself,
// or nil (JSON null, meaning "remove the key") when empty.
func nullable(s string) any {
	if s == "" {
		return nil
	}

	return s
}
