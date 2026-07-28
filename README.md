# SyncWatch

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
  dropped stream self-heals.
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

See [docs/kubernetes-deployment.md](docs/kubernetes-deployment.md) for the intended
cluster setup: RBAC, an optional admission policy restricting SyncWatch's
write access to just the auto-sync field, ingress, and OIDC.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Asset attribution

Status icons are [Font Awesome Free](https://fontawesome.com/license/free)
5.15.4 solid glyphs (CC BY 4.0) — the same glyphs the
[Argo CD](https://argo-cd.readthedocs.io/) UI uses, in Argo CD's status
color palette. The Argo logo (`static/argo.svg`) is from the
[argoproj/argo-cd](https://github.com/argoproj/argo-cd) repository and is a
trademark of The Linux Foundation / CNCF, used here to identify the Argo CD
instance this tool operates on.
