package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

var applicationGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

// Annotations set on Application resources to record why/when/by whom
// auto-sync was paused. Notes live on the Application itself so they survive
// restarts without separate storage and update live via the same watch.
const (
	annNote     = "snarlysodboxer.github.io/syncwatch-note"
	annPausedBy = "snarlysodboxer.github.io/syncwatch-paused-by"
	annPausedAt = "snarlysodboxer.github.io/syncwatch-paused-at"
)

// AppView is the per-application state shown in the UI.
type AppView struct {
	Name     string `json:"name"`
	Project  string `json:"project"`
	Sync     string `json:"sync"`   // Synced | OutOfSync | Unknown
	Health   string `json:"health"` // Healthy | Progressing | Degraded | Suspended | Missing | Unknown
	Syncing  bool   `json:"syncing"`
	AutoSync bool   `json:"autoSync"`
	Prune    bool   `json:"prune"`
	SelfHeal bool   `json:"selfHeal"`
	Note     string `json:"note"`
	PausedBy string `json:"pausedBy"`
	PausedAt string `json:"pausedAt"`
}

// Hub fans out server-sent events to connected browsers.
type Hub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

// NewHub returns an empty Hub ready for subscribers.
func NewHub() *Hub {
	return &Hub{subs: make(map[chan []byte]struct{})}
}

// Subscribe registers and returns a new buffered event channel.
func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subs[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a channel returned by Subscribe.
func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs, ch)
}

// Broadcast sends one formatted SSE event to every subscriber, dropping it
// for subscribers whose buffers are full rather than blocking.
func (h *Hub) Broadcast(event string, payload any) {
	msg := formatSSE(event, payload)
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
			slog.Warn("dropping event for slow SSE subscriber", "event", event)
		}
	}
}

func formatSSE(event string, payload any) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal SSE payload", "error", err)
		return nil
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", event)
	fmt.Fprintf(&buf, "data: %s\n\n", data)
	return buf.Bytes()
}

// Store holds the current view of all applications and broadcasts changes.
type Store struct {
	mu   sync.RWMutex
	apps map[string]AppView
	hub  *Hub
}

// NewStore returns an empty Store that broadcasts changes to hub.
func NewStore(hub *Hub) *Store {
	return &Store{apps: make(map[string]AppView), hub: hub}
}

// Upsert stores an application view, broadcasting it if anything changed.
func (s *Store) Upsert(app AppView) {
	s.mu.Lock()
	old, existed := s.apps[app.Name]
	s.apps[app.Name] = app
	s.mu.Unlock()
	if !existed || old != app {
		s.hub.Broadcast("app", app)
	}
}

// Delete removes an application, broadcasting the removal if it existed.
func (s *Store) Delete(name string) {
	s.mu.Lock()
	_, existed := s.apps[name]
	delete(s.apps, name)
	s.mu.Unlock()
	if existed {
		s.hub.Broadcast("delete", map[string]string{"name": name})
	}
}

// Get returns the view of one application, if known.
func (s *Store) Get(name string) (AppView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[name]
	return app, ok
}

// Snapshot returns all application views, sorted by name.
func (s *Store) Snapshot() []AppView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	apps := make([]AppView, 0, len(s.apps))
	for _, app := range s.apps {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps
}

// Janitor reconciles pause bookkeeping when auto-sync is changed by tools
// other than SyncWatch (the ArgoCD UI/CLI, kubectl, ...).
type Janitor interface {
	ClearPauseAnnotations(ctx context.Context, name string) error
	StampExternalPause(ctx context.Context, name, who string) error
}

// StartWatch runs a shared informer on Application resources and feeds the store.
func StartWatch(ctx context.Context, client dynamic.Interface, namespace string, store *Store, janitor Janitor) error {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 10*time.Minute, namespace, nil)
	informer := factory.ForResource(applicationGVR).Informer()

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if u, ok := obj.(*unstructured.Unstructured); ok {
				app := parseApp(u)
				store.Upsert(app)
				go reconcilePauseMeta(ctx, janitor, nil, app, u)
			}
		},
		UpdateFunc: func(oldObj, obj any) {
			if u, ok := obj.(*unstructured.Unstructured); ok {
				app := parseApp(u)
				store.Upsert(app)
				if oldU, ok := oldObj.(*unstructured.Unstructured); ok {
					old := parseApp(oldU)
					go reconcilePauseMeta(ctx, janitor, &old, app, u)
				}
			}
		},
		DeleteFunc: func(obj any) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			if u, ok := obj.(*unstructured.Unstructured); ok {
				store.Delete(u.GetName())
			}
		},
	})
	if err != nil {
		return err
	}

	go informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("timed out waiting for Application informer to sync")
	}
	slog.Info("application informer synced", "apps", len(store.Snapshot()))
	return nil
}

func parseApp(u *unstructured.Unstructured) AppView {
	app := AppView{
		Name:    u.GetName(),
		Sync:    "Unknown",
		Health:  "Unknown",
		Project: nestedString(u, "spec", "project"),
	}
	if s := nestedString(u, "status", "sync", "status"); s != "" {
		app.Sync = s
	}
	if h := nestedString(u, "status", "health", "status"); h != "" {
		app.Health = h
	}
	app.Syncing = nestedString(u, "status", "operationState", "phase") == "Running"

	// Auto-sync is on when spec.syncPolicy.automated exists, unless ArgoCD
	// v3's explicit enabled field says otherwise.
	if automated, found, _ := unstructured.NestedMap(u.Object, "spec", "syncPolicy", "automated"); found {
		app.AutoSync = true
		if enabled, ok := automated["enabled"].(bool); ok && !enabled {
			app.AutoSync = false
		}
		app.Prune, _ = automated["prune"].(bool)
		app.SelfHeal, _ = automated["selfHeal"].(bool)
	}

	annotations := u.GetAnnotations()
	app.Note = annotations[annNote]
	app.PausedBy = annotations[annPausedBy]
	app.PausedAt = annotations[annPausedAt]
	return app
}

func nestedString(u *unstructured.Unstructured, fields ...string) string {
	s, _, _ := unstructured.NestedString(u.Object, fields...)
	return s
}

// reconcilePauseMeta keeps the pause annotations truthful when auto-sync is
// changed by tools other than SyncWatch. SyncWatch's own toggles set spec and
// annotations in one atomic patch, so anything inconsistent here was done
// externally (ArgoCD UI/CLI, kubectl, ...). Both repairs are idempotent, and
// the patches they issue produce consistent states, so this cannot loop.
func reconcilePauseMeta(ctx context.Context, janitor Janitor, old *AppView, app AppView, u *unstructured.Unstructured) {
	if janitor == nil {
		return
	}
	switch {
	// Resumed externally: pause bookkeeping is stale — clear it. Also covers
	// startup (old == nil), catching resumes that happened while we were down.
	case app.AutoSync && (app.Note != "" || app.PausedBy != "" || app.PausedAt != ""):
		slog.Info("auto-sync was re-enabled outside syncwatch, clearing stale pause annotations", "app", app.Name)
		if err := janitor.ClearPauseAnnotations(ctx, app.Name); err != nil {
			slog.Error("failed to clear stale pause annotations", "app", app.Name, "error", err)
		}
	// Paused externally: we watched auto-sync turn off with no paused-at
	// stamped, so record when and (best-effort) by what.
	case old != nil && old.AutoSync && !app.AutoSync && app.PausedAt == "":
		who := externalActor(u)
		slog.Info("auto-sync was paused outside syncwatch", "app", app.Name, "by", who)
		if err := janitor.StampExternalPause(ctx, app.Name, who); err != nil {
			slog.Error("failed to record external pause", "app", app.Name, "error", err)
		}
	}
}

// externalActor guesses which client last touched spec.syncPolicy.automated,
// from the field managers Kubernetes tracks in metadata.managedFields (the
// ArgoCD API server, kubectl, etc. each write under their own manager name).
func externalActor(u *unstructured.Unstructured) string {
	manager := ""
	var latest time.Time
	for _, mf := range u.GetManagedFields() {
		if mf.Manager == "syncwatch" || mf.FieldsV1 == nil {
			continue
		}
		// Cheap containment test instead of walking the fieldsV1 tree: the
		// key appears iff this manager owns something under automated.
		if !bytes.Contains(mf.FieldsV1.Raw, []byte(`"f:automated"`)) {
			continue
		}
		if mf.Time != nil && mf.Time.Time.After(latest) {
			latest = mf.Time.Time
			manager = mf.Manager
		}
	}
	switch {
	case manager == "":
		return "an external tool"
	case strings.Contains(manager, "argocd"):
		return "the ArgoCD UI/CLI"
	default:
		return manager // e.g. "kubectl-edit"
	}
}
