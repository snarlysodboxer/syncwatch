"use strict";

// Font Awesome Free 5.15.4 solid icon paths (CC BY 4.0), the same glyphs the
// Argo CD UI uses for sync and health status.
const ICONS = {
  "check-circle": { vb: "0 0 512 512", d: "M504 256c0 136.967-111.033 248-248 248S8 392.967 8 256 119.033 8 256 8s248 111.033 248 248zM227.314 387.314l184-184c6.248-6.248 6.248-16.379 0-22.627l-22.627-22.627c-6.248-6.249-16.379-6.249-22.628 0L216 308.118l-70.059-70.059c-6.248-6.248-16.379-6.248-22.628 0l-22.627 22.627c-6.248 6.248-6.248 16.379 0 22.627l104 104c6.249 6.249 16.379 6.249 22.628.001z" },
  "arrow-alt-circle-up": { vb: "0 0 512 512", d: "M8 256C8 119 119 8 256 8s248 111 248 248-111 248-248 248S8 393 8 256zm292 116V256h70.9c10.7 0 16.1-13 8.5-20.5L264.5 121.2c-4.7-4.7-12.2-4.7-16.9 0l-115 114.3c-7.6 7.6-2.2 20.5 8.5 20.5H212v116c0 6.6 5.4 12 12 12h64c6.6 0 12-5.4 12-12z" },
  "circle-notch": { vb: "0 0 512 512", d: "M288 39.056v16.659c0 10.804 7.281 20.159 17.686 23.066C383.204 100.434 440 171.518 440 256c0 101.689-82.295 184-184 184-101.689 0-184-82.295-184-184 0-84.47 56.786-155.564 134.312-177.219C216.719 75.874 224 66.517 224 55.712V39.064c0-15.709-14.834-27.153-30.046-23.234C86.603 43.482 7.394 141.206 8.003 257.332c.72 137.052 111.477 246.956 248.531 246.667C393.255 503.711 504 392.788 504 256c0-115.633-79.14-212.779-186.211-240.236C302.678 11.889 288 23.456 288 39.056z" },
  "heart": { vb: "0 0 512 512", d: "M462.3 62.6C407.5 15.9 326 24.3 275.7 76.2L256 96.5l-19.7-20.3C186.1 24.3 104.5 15.9 49.7 62.6c-62.8 53.6-66.1 149.8-9.9 207.9l193.5 199.8c12.5 12.9 32.8 12.9 45.3 0l193.5-199.8c56.3-58.1 53-154.3-9.8-207.9z" },
  "heart-broken": { vb: "0 0 512 512", d: "M473.7 73.8l-2.4-2.5c-46-47-118-51.7-169.6-14.8L336 159.9l-96 64 48 128-144-144 96-64-28.6-86.5C159.7 19.6 87 24 40.7 71.4l-2.4 2.4C-10.4 123.6-12.5 202.9 31 256l212.1 218.6c7.1 7.3 18.6 7.3 25.7 0L481 255.9c43.5-53 41.4-132.3-7.3-182.1z" },
  "pause-circle": { vb: "0 0 512 512", d: "M256 8C119 8 8 119 8 256s111 248 248 248 248-111 248-248S393 8 256 8zm-16 328c0 8.8-7.2 16-16 16h-48c-8.8 0-16-7.2-16-16V176c0-8.8 7.2-16 16-16h48c8.8 0 16 7.2 16 16v160zm112 0c0 8.8-7.2 16-16 16h-48c-8.8 0-16-7.2-16-16V176c0-8.8 7.2-16 16-16h48c8.8 0 16 7.2 16 16v160z" },
  "ghost": { vb: "0 0 384 512", d: "M186.1.09C81.01 3.24 0 94.92 0 200.05v263.92c0 14.26 17.23 21.39 27.31 11.31l24.92-18.53c6.66-4.95 16-3.99 21.51 2.21l42.95 48.35c6.25 6.25 16.38 6.25 22.63 0l40.72-45.85c6.37-7.17 17.56-7.17 23.92 0l40.72 45.85c6.25 6.25 16.38 6.25 22.63 0l42.95-48.35c5.51-6.2 14.85-7.17 21.51-2.21l24.92 18.53c10.08 10.08 27.31 2.94 27.31-11.31V192C384 84 294.83-3.17 186.1.09zM128 224c-17.67 0-32-14.33-32-32s14.33-32 32-32 32 14.33 32 32-14.33 32-32 32zm128 0c-17.67 0-32-14.33-32-32s14.33-32 32-32 32 14.33 32 32-14.33 32-32 32z" },
  "question-circle": { vb: "0 0 512 512", d: "M504 256c0 136.997-111.043 248-248 248S8 392.997 8 256C8 119.083 119.043 8 256 8s248 111.083 248 248zM262.655 90c-54.497 0-89.255 22.957-116.549 63.758-3.536 5.286-2.353 12.415 2.715 16.258l34.699 26.31c5.205 3.947 12.621 3.008 16.665-2.122 17.864-22.658 30.113-35.797 57.303-35.797 20.429 0 45.698 13.148 45.698 32.958 0 14.976-12.363 22.667-32.534 33.976C247.128 238.528 216 254.941 216 296v4c0 6.627 5.373 12 12 12h56c6.627 0 12-5.373 12-12v-1.333c0-28.462 83.186-29.647 83.186-106.667 0-58.002-60.165-102-116.531-102zM256 338c-25.365 0-46 20.635-46 46 0 25.364 20.635 46 46 46s46-20.636 46-46c0-25.365-20.635-46-46-46z" },
};

const SYNC_STATUS = {
  Synced: { icon: "check-circle", cls: "st-success" },
  OutOfSync: { icon: "arrow-alt-circle-up", cls: "st-warning" },
  Unknown: { icon: "circle-notch", cls: "st-unknown" },
};

const HEALTH_STATUS = {
  Healthy: { icon: "heart", cls: "st-success" },
  Progressing: { icon: "circle-notch", cls: "st-running", spin: true },
  Degraded: { icon: "heart-broken", cls: "st-failed" },
  Suspended: { icon: "pause-circle", cls: "st-suspended" },
  Missing: { icon: "ghost", cls: "st-warning" },
  Unknown: { icon: "question-circle", cls: "st-unknown" },
};

const state = new Map(); // name -> app
const rows = new Map();  // name -> {root, name, project, sync, health, note, meta, toggle}
const pending = new Map(); // name -> timeout id
let focusNoteFor = null;

const appsEl = document.getElementById("apps");
const summaryEl = document.getElementById("summary");
const emptyEl = document.getElementById("empty");

function svgIcon(name, spin) {
  const icon = ICONS[name];
  return `<svg viewBox="${icon.vb}" class="${spin ? "spin" : ""}" aria-hidden="true"><path d="${icon.d}"/></svg>`;
}

function setStatus(cell, def, label, syncing) {
  const spin = def.spin || false;
  cell.className = `status ${def.cls}`;
  cell.innerHTML = svgIcon(def.icon, spin) + `<span class="label">${label}</span>` +
    (syncing ? `<span class="st-running" title="sync operation running">${svgIcon("circle-notch", true)}</span>` : "");
}

function relTime(iso) {
  if (!iso) return "";
  const secs = Math.max(0, (Date.now() - Date.parse(iso)) / 1000);
  if (secs < 90) return "just now";
  if (secs < 5400) return `${Math.round(secs / 60)}m ago`;
  if (secs < 129600) return `${Math.round(secs / 3600)}h ago`;
  return `${Math.round(secs / 86400)}d ago`;
}

function buildRow(app) {
  const root = document.createElement("div");
  root.className = "app-row";

  const nameCell = document.createElement("div");
  const nameEl = document.createElement("div");
  nameEl.className = "app-name";
  const projectEl = document.createElement("div");
  projectEl.className = "app-project";
  nameCell.append(nameEl, projectEl);

  const syncCell = document.createElement("div");
  const healthCell = document.createElement("div");

  const noteCell = document.createElement("div");
  noteCell.className = "note-cell";
  const noteInput = document.createElement("input");
  noteInput.className = "note-input";
  noteInput.type = "text";
  noteInput.maxLength = 200;
  noteInput.placeholder = "why is sync paused?";
  const noteMeta = document.createElement("div");
  noteMeta.className = "note-meta";
  const noteNone = document.createElement("span");
  noteNone.className = "note-none";
  noteNone.textContent = "—";
  noteCell.append(noteInput, noteMeta, noteNone);

  noteInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") noteInput.blur();
    if (e.key === "Escape") {
      noteInput.value = state.get(app.name)?.note || "";
      noteInput.blur();
    }
  });
  noteInput.addEventListener("blur", () => {
    const current = state.get(app.name);
    if (current && noteInput.value !== current.note) {
      post(`/api/apps/${encodeURIComponent(app.name)}/note`, { note: noteInput.value })
        .catch(() => {});
    }
    render(); // apply any reorder deferred while typing
  });

  const toggleCell = document.createElement("div");
  toggleCell.className = "toggle-cell";
  const switchLabel = document.createElement("label");
  switchLabel.className = "switch";
  const toggle = document.createElement("input");
  toggle.type = "checkbox";
  const slider = document.createElement("span");
  slider.className = "slider";
  switchLabel.append(toggle, slider);
  toggleCell.append(switchLabel);

  toggle.addEventListener("change", () => {
    const enabled = toggle.checked;
    markPending(app.name);
    if (!enabled) focusNoteFor = app.name;
    post(`/api/apps/${encodeURIComponent(app.name)}/autosync`, { enabled, note: "" })
      .catch(() => {
        clearPending(app.name);
        render();
      });
  });

  root.append(nameCell, syncCell, healthCell, noteCell, toggleCell);
  return { root, name: nameEl, project: projectEl, sync: syncCell, health: healthCell, noteInput, noteMeta, noteNone, toggle, switchLabel };
}

function updateRow(app) {
  let row = rows.get(app.name);
  if (!row) {
    row = buildRow(app);
    rows.set(app.name, row);
    appsEl.append(row.root);
  }
  row.root.classList.toggle("paused", !app.autoSync);
  row.root.classList.toggle("pending", pending.has(app.name));
  row.name.textContent = app.name;
  row.project.textContent = app.project;

  setStatus(row.sync, SYNC_STATUS[app.sync] || SYNC_STATUS.Unknown, app.sync, app.syncing);
  setStatus(row.health, HEALTH_STATUS[app.health] || HEALTH_STATUS.Unknown, app.health, false);

  const showNote = !app.autoSync;
  row.noteInput.hidden = !showNote;
  row.noteMeta.hidden = !showNote;
  row.noteNone.hidden = showNote;
  if (showNote) {
    if (document.activeElement !== row.noteInput) row.noteInput.value = app.note || "";
    const parts = [];
    if (app.pausedBy) parts.push(`paused by ${app.pausedBy}`);
    if (app.pausedAt) parts.push(relTime(app.pausedAt));
    row.noteMeta.textContent = parts.join(" · ");
    row.noteMeta.title = app.pausedAt || "";
  }

  row.toggle.checked = app.autoSync;
  row.switchLabel.title = app.autoSync
    ? `auto-sync enabled (prune: ${app.prune}, selfHeal: ${app.selfHeal}) — click to pause`
    : "auto-sync paused — click to resume";

  if (focusNoteFor === app.name && !app.autoSync) {
    focusNoteFor = null;
    row.noteInput.focus();
  }
}

function render() {
  for (const app of state.values()) updateRow(app);
  for (const [name, row] of rows) {
    if (!state.has(name)) {
      row.root.remove();
      rows.delete(name);
    }
  }

  // Defer reordering while the user is typing in a note so the row (and
  // focus) doesn't jump out from under them.
  const typing = document.activeElement && appsEl.contains(document.activeElement);
  if (!typing) {
    const order = [...state.values()]
      .sort((a, b) => (a.autoSync - b.autoSync) || a.name.localeCompare(b.name));
    for (const app of order) appsEl.append(rows.get(app.name).root);
  }

  const total = state.size;
  const apps = [...state.values()];
  const paused = apps.filter((a) => !a.autoSync).length;
  const chips = [`<span class="chip">${total} apps</span>`];
  chips.push(paused
    ? `<span class="chip warn">${paused} paused</span>`
    : `<span class="chip good">auto-sync on everywhere</span>`);
  const addStat = (count, def, label) => {
    if (count) chips.push(
      `<span class="chip stat ${def.cls}" title="${count} ${label}">${svgIcon(def.icon, def.spin)}${count}</span>`);
  };
  addStat(apps.filter((a) => a.sync === "OutOfSync").length, SYNC_STATUS.OutOfSync, "out of sync");
  addStat(apps.filter((a) => a.sync === "Unknown").length, SYNC_STATUS.Unknown, "sync status unknown");
  for (const health of ["Progressing", "Suspended", "Degraded", "Missing", "Unknown"]) {
    addStat(apps.filter((a) => a.health === health).length, HEALTH_STATUS[health],
      health === "Unknown" ? "health unknown" : health.toLowerCase());
  }
  summaryEl.innerHTML = chips.join("");
  emptyEl.hidden = total > 0;
}

function markPending(name) {
  clearPending(name);
  pending.set(name, setTimeout(() => {
    clearPending(name);
    render();
    toast("no confirmation from server — change may not have applied");
  }, 8000));
  rows.get(name)?.root.classList.add("pending");
}

function clearPending(name) {
  const t = pending.get(name);
  if (t !== undefined) {
    clearTimeout(t);
    pending.delete(name);
  }
}

async function post(url, body) {
  let resp;
  try {
    resp = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      redirect: "manual",
    });
  } catch (err) {
    toast("request failed — checking connection");
    checkConnection();
    throw err;
  }
  if (resp.type === "opaqueredirect") {
    // The auth session expired and the proxy answered with a redirect to
    // the identity provider — reload the page through the login flow.
    checkConnection();
    throw new Error(`POST ${url}: auth redirect`);
  }
  if (!resp.ok) {
    toast(`request failed: ${(await resp.text()).trim() || resp.status}`);
    throw new Error(`POST ${url}: ${resp.status}`);
  }
}

let toastTimer;
function toast(msg) {
  const el = document.getElementById("toast");
  el.textContent = msg;
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, 5000);
}

// ---- live event stream ----

const connEl = document.getElementById("conn");
const connLabel = document.getElementById("conn-label");
const STALE_AFTER_MS = 70000; // the server pings every 25s; 2+ missed = dead
let es;
let lastEventAt = Date.now();
let probing = false;

function setConn(cls, label) {
  connEl.className = `conn conn-${cls}`;
  connLabel.textContent = label;
  document.body.classList.toggle("disconnected", cls === "down");
}

function touch() { lastEventAt = Date.now(); }

function connect() {
  if (es) es.close();
  es = new EventSource("/api/events");
  es.addEventListener("open", () => {
    touch();
    setConn("live", "Live");
  });
  es.addEventListener("ping", touch);
  es.addEventListener("snapshot", (e) => {
    touch();
    state.clear();
    for (const app of JSON.parse(e.data)) state.set(app.name, app);
    pending.forEach((_, name) => clearPending(name));
    render();
  });
  es.addEventListener("app", (e) => {
    touch();
    const app = JSON.parse(e.data);
    state.set(app.name, app);
    clearPending(app.name);
    render();
  });
  es.addEventListener("delete", (e) => {
    touch();
    state.delete(JSON.parse(e.data).name);
    render();
  });
  es.addEventListener("error", () => {
    setConn("down", "Connection lost");
    // EventSource retries transient failures itself, but a reconnect that
    // gets redirected to the identity provider is fatal to it (readyState
    // CLOSED, no further retries) — probe right away instead of waiting for
    // the watchdog.
    if (es.readyState === EventSource.CLOSED) checkConnection();
  });
}

// checkConnection distinguishes a dead stream from an expired auth session:
// if /healthz still answers, the stream is rebuilt in place; if the request
// gets redirected (to the identity provider) the whole page reloads so the
// browser re-runs the login flow. Network errors keep the banner up and let
// the watchdog try again.
async function checkConnection() {
  if (probing) return;
  probing = true;
  try {
    const resp = await fetch("/healthz", { redirect: "manual", cache: "no-store" });
    if (resp.ok) {
      touch();
      connect();
    } else {
      location.reload();
    }
  } catch {
    // server or network down — stay disconnected, watchdog retries
  } finally {
    probing = false;
  }
}

// Watchdog for connections that die *silently* (proxy/NAT drops with no FIN):
// no error event ever fires for those, the page just stops receiving pings.
function watchdogCheck() {
  if (es.readyState === EventSource.CLOSED || Date.now() - lastEventAt > STALE_AFTER_MS) {
    setConn("down", "Connection lost");
    checkConnection();
  }
}
setInterval(watchdogCheck, 15000);
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) watchdogCheck();
});

connect();
setInterval(() => { // keep "Nm ago" labels fresh
  if (!(document.activeElement && appsEl.contains(document.activeElement))) render();
}, 60000);
