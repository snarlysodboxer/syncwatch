package main

import (
	"context"
	"time"
)

// startDemo seeds the store with fake applications covering every status
// combination and returns a Patcher that mutates them in memory, so the UI
// can be developed and previewed without a cluster.
func startDemo(ctx context.Context, store *Store, devUser string) Patcher {
	if devUser == "" {
		devUser = "demo@example.com"
	}
	seed := []AppView{
		{Name: "alloy", Project: "demo", Sync: "Synced", Health: "Healthy", AutoSync: false, Prune: true, SelfHeal: true, Note: "tuning log pipeline (da)", PausedBy: devUser, PausedAt: "2026-07-24T09:15:00Z"},
		{Name: "argocd", Project: "demo", Sync: "Synced", Health: "Healthy", AutoSync: true, Prune: true, SelfHeal: true},
		{Name: "cert-manager", Project: "demo", Sync: "OutOfSync", Health: "Healthy", AutoSync: false, Prune: true, SelfHeal: true, PausedBy: devUser, PausedAt: "2026-07-25T18:40:00Z"},
		{Name: "envoy-gateway", Project: "demo", Sync: "Synced", Health: "Healthy", AutoSync: true, Prune: true, SelfHeal: true},
		{Name: "grafana", Project: "demo", Sync: "OutOfSync", Health: "Progressing", Syncing: true, AutoSync: true, Prune: true, SelfHeal: true},
		{Name: "istio", Project: "demo", Sync: "Synced", Health: "Degraded", AutoSync: true, Prune: true, SelfHeal: true},
		{Name: "kiali", Project: "demo", Sync: "Unknown", Health: "Unknown", AutoSync: true},
		{Name: "loki", Project: "demo", Sync: "Synced", Health: "Suspended", AutoSync: true, Prune: true},
		{Name: "mimir", Project: "demo", Sync: "OutOfSync", Health: "Missing", AutoSync: false, Note: "improving RBAC (jl)", PausedBy: "jane@example.com", PausedAt: "2026-07-20T14:00:00Z"},
		{Name: "team-access", Project: "demo", Sync: "Synced", Health: "Healthy", AutoSync: true, Prune: true, SelfHeal: true},
	}
	for _, app := range seed {
		store.Upsert(app)
	}

	// Cycle one app through a fake sync so live updates are visible.
	go func() {
		states := []struct {
			sync, health string
			syncing      bool
		}{
			{"OutOfSync", "Healthy", false},
			{"OutOfSync", "Progressing", true},
			{"Synced", "Progressing", true},
			{"Synced", "Healthy", false},
		}
		i := 0
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app, ok := store.Get("envoy-gateway")
				if !ok {
					continue
				}
				s := states[i%len(states)]
				app.Sync, app.Health, app.Syncing = s.sync, s.health, s.syncing
				store.Upsert(app)
				i++
			}
		}
	}()

	return &DemoPatcher{store: store}
}

// DemoPatcher implements Patcher against the in-memory store, for --demo mode.
type DemoPatcher struct {
	store *Store
}

// SetAutoSync flips auto-sync on the in-memory view.
func (p *DemoPatcher) SetAutoSync(_ context.Context, name string, enabled bool, note, who string) error {
	app, ok := p.store.Get(name)
	if !ok {
		return nil
	}
	app.AutoSync = enabled
	if enabled {
		app.Note, app.PausedBy, app.PausedAt = "", "", ""
	} else {
		app.Note = note
		app.PausedBy = who
		app.PausedAt = time.Now().UTC().Format(time.RFC3339)
	}
	p.store.Upsert(app)

	return nil
}

// SetNote sets the note on the in-memory view.
func (p *DemoPatcher) SetNote(_ context.Context, name, note string) error {
	app, ok := p.store.Get(name)
	if !ok {
		return nil
	}
	app.Note = note
	p.store.Upsert(app)

	return nil
}
