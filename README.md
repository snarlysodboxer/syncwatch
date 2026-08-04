# SyncWatch

<img src="static/favicon.svg" align="right" width="110" alt="SyncWatch logo">

A tiny live dashboard for ArgoCD auto-sync state.

## Why

A common development workflow pauses ArgoCD auto-sync on an application while
iterating on its manifests (`kustomize build ... | kubectl apply -f -`), then
re-enables it once the changes are merged. The failure mode is forgetting the
last step - and the ArgoCD UI has no at-a-glance view of which applications
have auto-sync paused. SyncWatch shows one row per ArgoCD Application with
its sync status, health status, and an auto-sync toggle, plus a note
explaining *why* sync is paused, *who* paused it, and *when*. Everything
updates live in every open browser - no refreshing.

## How it works

- **No ArgoCD API involved.** SyncWatch watches `Application` custom
  resources directly via the Kubernetes API with a shared informer - the same
  data ArgoCD's own API serves, with subscribe semantics for free.
- **Pause/resume** is a JSON merge patch flipping
  `spec.syncPolicy.automated.enabled` (ArgoCD v3 semantics: prune/selfHeal
  settings are preserved while paused).
- **Notes need no database.** They are annotations on the Application itself
  (`snarlysodboxer.github.io/syncwatch-note`, `.../syncwatch-paused-by`,
  `.../syncwatch-paused-at`): they survive restarts, travel with the app, are
  visible in `kubectl describe` and the ArgoCD UI, and annotation writes fire the
  same watch that powers live updates. Resuming auto-sync clears them.
- **Other tools stay in the loop.** SyncWatch doesn't own auto-sync: if
  someone resumes via the ArgoCD UI/CLI or kubectl, the watch notices and
  clears the now-stale note annotations; if someone *pauses* externally, it
  stamps `paused-at` and a best-effort `paused-by` (e.g. "the ArgoCD
  UI/CLI", derived from the field manager Kubernetes records in
  `metadata.managedFields`).
- **Live updates** reach browsers over server-sent events (`/api/events`):
  a `snapshot` event on connect, then `app` / `delete` events. `EventSource`
  reconnects automatically; every reconnect re-sends the snapshot, so a
  dropped stream self-heals. The server also sends a `ping` every 25s: a
  watchdog treats missing pings as a dead connection (proxies can drop
  streams without telling the browser), shows a banner, dims the stale data,
  and probes the server — a dead stream is rebuilt in place, while an
  expired auth session reloads the page back through the login flow.
- **Who paused it**: SyncWatch has no authentication of its own - it is
  designed to sit behind an authenticating proxy and records whatever
  identity the proxy forwards (see
  [Identity](#identity-who-paused-it) below). Outside the cluster it falls
  back to `--dev-user` or `$USER`.

## Development

```sh
nix develop            # go, gopls, gotools, staticcheck

go run . --demo        # fake data, no cluster needed — http://localhost:8080
go run .               # real data via $KUBECONFIG / ~/.kube/config (read + write!)
go run . -h            # all flags
```

`nix build` produces the static binary at `result/bin/syncwatch`.

## HTTP API

| Route | Description |
| --- | --- |
| `GET /` | the dashboard |
| `GET /api/events` | SSE stream: `snapshot`, then `app` / `delete` events |
| `POST /api/apps/{name}/autosync` | `{"enabled": bool, "note": "..."}` |
| `POST /api/apps/{name}/note` | `{"note": "..."}` (empty clears) |
| `GET /healthz` | liveness/readiness |

## Identity ("who paused it")

SyncWatch records the acting user from whatever the authenticating proxy in
front of it forwards, controlled by three flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--identity-header` | *(unset)* | request header carrying the identity, plaintext or JWT; checked first |
| `--identity-cookie` | `IdToken` | name prefix of cookies carrying an OIDC ID token JWT; empty disables |
| `--identity-claim` | `email` | JWT claim recorded as the user |

A `Bearer ` prefix on the header is stripped; a value that parses as a JWT
has the claim extracted; anything else is recorded verbatim. Starting points
for common proxies:

| Proxy | Flags |
| --- | --- |
| Envoy Gateway OIDC `SecurityPolicy` | works with the defaults (`IdToken-<suffix>` cookie) |
| oauth2-proxy | `--identity-header X-Forwarded-Email` |
| Google IAP | `--identity-header X-Goog-Authenticated-User-Email` |
| AWS ALB OIDC | `--identity-header x-amzn-oidc-data` |
| Cloudflare Access | `--identity-header Cf-Access-Authenticated-User-Email` |
| Authelia / authentik | `--identity-header Remote-Email` |
| IdPs without an email claim (e.g. some Keycloak setups) | add `--identity-claim preferred_username` |

Only the Envoy Gateway setup is regularly tested - PRs confirming or fixing
the others are very welcome.

Two things to know:

- JWT signatures are **not** verified. Run SyncWatch only behind the proxy,
  and make sure the proxy strips or overwrites these headers/cookies on
  inbound requests (most do by default) - otherwise authenticated users
  could spoof `paused-by`. It's an audit hint, not a security control,
  either way.
- The mechanism itself is easy to test without any proxy:
  `curl -X POST -H 'X-Forwarded-Email: dev@example.com' ...` against a
  locally running instance started with the matching flag.

## Deployment

Ready-to-fork Kustomize manifests live in [`kustomize/`](kustomize/):

- [`apps/syncwatch/base`](kustomize/apps/syncwatch/base) — the generic,
  ingress-agnostic core: Namespace, Deployment (nonroot, read-only root
  filesystem, tiny resource footprint), Service, RBAC (a Role in the
  `argocd` namespace: get/list/watch/patch on Applications), and an
  optional-but-recommended `ValidatingAdmissionPolicy` that rejects any
  write from SyncWatch's ServiceAccount other than flipping
  `spec.syncPolicy.automated.enabled` and setting its own annotations.
- [`apps/syncwatch/envoy`](kustomize/apps/syncwatch/envoy) — pulls in the
  base and adds Gateway API ingress: HTTPRoute, cert-manager Certificate +
  ReferenceGrant, Envoy Gateway OIDC `SecurityPolicy` (+ OAuth client
  Secret), a `BackendTrafficPolicy` disabling the request timeout (the SSE
  stream is one long-lived response — a gateway-level request timeout cuts
  it), and an Istio ambient `AuthorizationPolicy` restricting inbound
  traffic to the gateway (delete it and the `istio.io` namespace label if
  you don't run Istio). Using a different ingress or auth proxy? Compose
  your own variant on `base` the same way.
- [`overlays/prod/syncwatch`](kustomize/overlays/prod/syncwatch) — an
  example environment overlay: picks the `envoy` variant and patches in the
  hostname, certificate, and OIDC settings.

To deploy: copy the example overlay, replace the `example.com` / `CHANGEME`
values (real OAuth credentials belong in your secrets tooling — SOPS,
sealed-secrets, external-secrets, ...), point the Deployment at an image
you've built and pushed (`nix build` produces a static binary that runs
`FROM scratch`: no CA bundle or writable filesystem needed, run as e.g.
`USER 65534`), register the OIDC redirect URL on your OAuth client, and
`kubectl apply -k` it — or point ArgoCD at it, which is rather fitting.

After deploying: pause an unimportant app and confirm the row shows
"paused by \<your email\>" — this proves the ID token reaches the app (see
[Identity](#identity-who-paused-it) if it doesn't). Watch the pause appear
live from a second browser, then check the annotations landed:
`kubectl -n argocd get app <name> -o yaml | grep syncwatch`. To verify the
admission policy, try an annotation write it should reject:
`kubectl -n argocd annotate app <name> foo=bar --dry-run=server --as=system:serviceaccount:syncwatch:syncwatch`.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Asset attribution

Status icons are [Font Awesome Free](https://fontawesome.com/license/free)
5.15.4 solid glyphs (CC BY 4.0) — the same glyphs the
[Argo CD](https://argo-cd.readthedocs.io/) UI uses, in Argo CD's status
color palette. The SyncWatch logo (`static/favicon.svg`) is original to this
project.
