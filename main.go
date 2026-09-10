// SyncWatch is a single-page dashboard for ArgoCD auto-sync state.
//
// It watches ArgoCD Application resources via the Kubernetes API and serves
// a live-updating page showing each application's sync/health status and
// whether auto-sync is enabled, with the ability to pause/resume auto-sync
// and attach a short note explaining why sync is paused.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed static
var staticFiles embed.FS

func main() {
	var (
		listen         = flag.String("listen", ":8080", "address to serve HTTP on")
		namespace      = flag.String("namespace", "argocd", "namespace containing the ArgoCD Application resources")
		kubeconfig     = flag.String("kubeconfig", "", "path to a kubeconfig file (default: in-cluster config, falling back to $KUBECONFIG / ~/.kube/config)")
		devUser        = flag.String("dev-user", "", "identity to record for actions when the auth proxy provides none (default: $USER when running outside the cluster)")
		demo           = flag.Bool("demo", false, "serve fake data instead of connecting to a cluster (for UI development)")
		argoCDURL      = flag.String("argocd-url", "", "base URL of the ArgoCD UI (e.g. https://argocd.example.com); when set, application names link to it")
		identityHeader = flag.String("identity-header", "", "request header the auth proxy forwards the user's identity in, plaintext or JWT (e.g. X-Forwarded-Email, Authorization); checked before --identity-cookie")
		identityCookie = flag.String("identity-cookie", "IdToken", "name prefix of cookies holding an OIDC ID token JWT (default matches Envoy Gateway's IdToken-<suffix>); empty disables")
		identityClaim  = flag.String("identity-claim", "email", "JWT claim recorded as the acting user")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	argoBase, err := normalizeArgoCDURL(*argoCDURL)
	if err != nil {
		slog.Error("invalid --argocd-url", "value", *argoCDURL, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hub := NewHub()
	store := NewStore(hub)

	var patcher Patcher
	if *demo {
		slog.Info("running in demo mode, not connecting to any cluster")
		patcher = startDemo(ctx, store, *devUser)
		if *devUser == "" {
			*devUser = os.Getenv("USER")
		}
	} else {
		cfg, inCluster, err := loadRESTConfig(*kubeconfig)
		if err != nil {
			slog.Error("failed to load Kubernetes config", "error", err)
			os.Exit(1)
		}
		if !inCluster && *devUser == "" {
			*devUser = os.Getenv("USER")
		}
		client, err := dynamic.NewForConfig(cfg)
		if err != nil {
			slog.Error("failed to create Kubernetes client", "error", err)
			os.Exit(1)
		}
		k8sPatcher := &K8sPatcher{Client: client, Namespace: *namespace}
		if err := StartWatch(ctx, client, *namespace, store, k8sPatcher); err != nil {
			slog.Error("failed to start Application watch", "error", err)
			os.Exit(1)
		}
		patcher = k8sPatcher
	}

	server := &Server{Store: store, Hub: hub, Patcher: patcher, ArgoCDURL: argoBase, Identity: IdentityConfig{
		Header:       *identityHeader,
		CookiePrefix: *identityCookie,
		Claim:        *identityClaim,
		Fallback:     *devUser,
	}}

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /api/config", server.HandleConfig)
	mux.HandleFunc("GET /api/events", server.HandleEvents)
	mux.HandleFunc("POST /api/apps/{name}/autosync", server.HandleAutoSync)
	mux.HandleFunc("POST /api/apps/{name}/note", server.HandleNote)

	httpServer := &http.Server{Addr: *listen, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	slog.Info("serving", "listen", *listen, "namespace", *namespace, "demo", *demo, "argocdURL", argoBase)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
}

// normalizeArgoCDURL validates the configured ArgoCD base URL and strips any
// trailing slash so application links can be built by simple concatenation.
// The scheme is checked because this value ends up in an href: rejecting
// anything but http(s) at startup keeps a typo from becoming a javascript:
// link on every row.
func normalizeArgoCDURL(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("missing host")
	}

	return strings.TrimRight(u.String(), "/"), nil
}

// loadRESTConfig returns a Kubernetes REST config, preferring an explicit
// kubeconfig path, then in-cluster config, then the default kubeconfig
// loading rules ($KUBECONFIG or ~/.kube/config).
func loadRESTConfig(kubeconfig string) (cfg *rest.Config, inCluster bool, err error) {
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		return cfg, false, err
	}
	if cfg, err = rest.InClusterConfig(); err == nil {
		return cfg, true, nil
	}
	if !errors.Is(err, rest.ErrNotInCluster) {
		return nil, false, err
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).ClientConfig()

	return cfg, false, err
}
