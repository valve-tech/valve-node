// #/dash/<id> — live sync/peer/disk dashboard, fed by the target's 5s
// monitor SSE stream (see internal/monitor: the poller ticks every 5s
// server-side; this file just renders whatever it sends) — plus the day-2
// operator controls added in v0.2: service start/stop/restart, per-service
// clear-and-resync (behind a typed-confirm modal), disk usage/size
// estimates, and endpoint reachability.
import * as api from "./api";
import { badge, copyToClipboard, dot, escapeHtml, fmtBytes, fmtDuration, fmtInt, fmtPct, footer } from "./ui";

// highDiskUsagePct is the threshold above which the disk card switches from
// its normal styling to a warning one.
const highDiskUsagePct = 85;

const SERVICE_LABEL: Record<api.ServiceID, string> = { exec: "Execution", beacon: "Beacon" };

export function renderDashboard(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let streamStop: (() => void) | null = null;
  let prevSnapshot: api.Snapshot | null = null;
  let latestSnapshot: api.Snapshot | null = null;
  let execBlocksPerSec: number | null = null;

  let du: api.DiskUsage | null = null;
  let duErr: string | null = null;
  let endpoints: api.EndpointInfo | null = null;
  let endpointsErr: string | null = null;

  // pending tracks an in-flight start/stop/restart per service, so the
  // matching button can show a spinner and every button for that service
  // can be disabled until the request resolves. Actual active/inactive
  // state always comes from the monitor SSE stream, never from this map or
  // from the action response — the response is only used to detect failure.
  const pending: Record<api.ServiceID, api.ServiceActionKind | null> = { exec: null, beacon: null };
  let actionErr: string | null = null;

  root.innerHTML = `<h1>Dashboard: ${escapeHtml(targetId)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#dash-body")!;
  const footerEl = root.querySelector<HTMLElement>("#dash-footer")!;

  body.addEventListener("click", (ev) => {
    const target = (ev.target as HTMLElement).closest<HTMLElement>("[data-action]");
    if (!target || !body.contains(target)) return;
    const action = target.dataset.action;
    if (action === "svc-action") {
      const svc = target.dataset.svc as api.ServiceID | undefined;
      const kind = target.dataset.kind as api.ServiceActionKind | undefined;
      if (svc && kind) void runServiceAction(svc, kind);
    } else if (action === "open-clear") {
      const svc = target.dataset.svc as api.ServiceID | undefined;
      if (svc) openClearModal(svc);
    } else if (action === "copy") {
      const text = target.dataset.copy;
      if (text) void copyButton(target, text);
    } else if (action === "retry-du") {
      void loadDiskUsage();
    } else if (action === "retry-endpoints") {
      void loadEndpoints();
    }
  });

  init();

  async function init(): Promise<void> {
    let target: api.Target | undefined;
    let catalog: api.Catalog | undefined;
    try {
      const [targets, cat] = await Promise.all([api.listTargets(), api.getCatalog()]);
      target = targets.find((t) => t.id === targetId);
      catalog = cat;
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load target: ${escapeHtml(String(err))}</p>`;
      return;
    }
    if (disposed) return;

    if (!target) {
      body.innerHTML = `<p class="error">Target "${escapeHtml(targetId)}" not found. <a href="#/targets">Back to targets</a></p>`;
      return;
    }
    if (!target.wire) {
      body.innerHTML = `<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(targetId)}">Run the setup wizard →</a></p>`;
      return;
    }

    const net = catalog?.networks.find((n) => n.ChainID === target!.wire!.ChainID);
    if (net) footerEl.innerHTML = footer(net.Name, net.LearnURL);

    body.innerHTML = `<p class="muted">Connecting…</p>`;
    streamStop = api.streamMonitor(targetId, (snap) => {
      if (disposed) return;
      updateRate(snap);
      prevSnapshot = snap;
      latestSnapshot = snap;
      render();
    });

    void loadDiskUsage();
    void loadEndpoints();
  }

  async function loadDiskUsage(): Promise<void> {
    duErr = null;
    try {
      du = await api.getDiskUsage(targetId);
    } catch (err) {
      du = null;
      duErr = String(err instanceof Error ? err.message : err);
    }
    if (!disposed) render();
  }

  async function loadEndpoints(): Promise<void> {
    endpointsErr = null;
    try {
      endpoints = await api.getEndpoints(targetId);
    } catch (err) {
      endpoints = null;
      endpointsErr = String(err instanceof Error ? err.message : err);
    }
    if (!disposed) render();
  }

  function updateRate(snap: api.Snapshot): void {
    if (!prevSnapshot) return;
    const deltaSeconds = (new Date(snap.at).getTime() - new Date(prevSnapshot.at).getTime()) / 1000;
    const deltaBlocks = snap.execHead - prevSnapshot.execHead;
    if (deltaSeconds > 0 && deltaBlocks >= 0) {
      const rate = deltaBlocks / deltaSeconds;
      // Simple exponential smoothing so one slow/fast tick doesn't swing the
      // ETA wildly.
      execBlocksPerSec = execBlocksPerSec === null ? rate : execBlocksPerSec * 0.7 + rate * 0.3;
    }
  }

  function render(): void {
    if (!latestSnapshot) return;
    const snap = latestSnapshot;
    body.innerHTML = `
      <div class="card-grid">
        ${execSyncCard(snap)}
        ${beaconSyncCard(snap)}
        ${peersCard(snap)}
        ${diskCard(snap)}
        ${storageCard(snap)}
        ${endpointsCard()}
        ${servicesCard(snap)}
      </div>
      <p class="muted small">Last updated ${escapeHtml(new Date(snap.at).toLocaleTimeString())}</p>
    `;
  }

  // syncETA computes the execution head's lag behind the reference head and
  // a human ETA at the current smoothed rate (execBlocksPerSec, updated by
  // updateRate on every monitor tick). Shared by the Execution-sync card
  // and the Storage card, so both surface the same rate-based estimate
  // instead of drifting out of sync with each other.
  function syncETA(snap: api.Snapshot): { lag: number | null; eta: string } {
    const hasRef = snap.refHead > 0;
    const lag = hasRef ? snap.refHead - snap.execHead : null;
    const eta =
      lag !== null && lag > 0 && execBlocksPerSec && execBlocksPerSec > 0
        ? fmtDuration(lag / execBlocksPerSec)
        : lag !== null && lag <= 0
          ? "caught up"
          : "—";
    return { lag, eta };
  }

  function execSyncCard(snap: api.Snapshot): string {
    const { lag, eta } = syncETA(snap);

    return `
      <div class="card">
        <h3>Execution sync</h3>
        <p>${snap.execSyncing ? badge("syncing", "warn") : badge("synced", "ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${fmtInt(snap.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${lag !== null ? fmtInt(snap.refHead) : "unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${lag !== null ? fmtInt(Math.max(lag, 0)) + " blocks" : "—"}</dd></div>
          <div><dt>ETA</dt><dd>${eta}</dd></div>
        </dl>
      </div>
    `;
  }

  function beaconSyncCard(snap: api.Snapshot): string {
    return `
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${snap.beaconDistance === 0 ? badge("synced", "ok") : badge("syncing", "warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${fmtInt(snap.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${fmtInt(snap.beaconDistance)}</dd></div>
        </dl>
      </div>
    `;
  }

  function peersCard(snap: api.Snapshot): string {
    return `
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${fmtInt(snap.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${fmtInt(snap.beaconPeers)}</dd></div>
        </dl>
      </div>
    `;
  }

  function diskCard(snap: api.Snapshot): string {
    const warn = snap.diskUsedPct >= highDiskUsagePct;
    return `
      <div class="card ${warn ? "card-warn" : ""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${warn ? "meter-warn" : ""}" style="width:${Math.min(snap.diskUsedPct, 100)}%"></div></div>
        <p>${fmtPct(snap.diskUsedPct)} used</p>
      </div>
    `;
  }

  // storageCard shows per-service current-vs-expected disk usage. These are
  // estimates ported from learn.valve.city's snapshot table, not live
  // measurements of the eventual synced size — labeled as such per the spec.
  // Per spec §3, it also surfaces the same rate-based ETA the
  // Execution-sync card computes (execBlocksPerSec) alongside the
  // current-vs-expected bar — only while the exec head is actually
  // advancing; it's omitted (not shown as "—" or "caught up") once there's
  // no meaningful rate to estimate from, since a stalled or already-synced
  // node has no "time remaining" to speak of here.
  function storageCard(snap: api.Snapshot): string {
    if (duErr) {
      return `
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${escapeHtml(duErr)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;
    }
    if (!du) {
      return `<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>`;
    }
    const execPct = du.ExpectedExecBytes > 0 ? Math.min((du.ExecBytes / du.ExpectedExecBytes) * 100, 100) : 0;
    const beaconPct = du.ExpectedBeaconBytes > 0 ? Math.min((du.BeaconBytes / du.ExpectedBeaconBytes) * 100, 100) : 0;
    const { lag, eta } = syncETA(snap);
    const advancing = lag !== null && lag > 0 && execBlocksPerSec !== null && execBlocksPerSec > 0;
    return `
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${fmtBytes(du.ExecBytes)} of ~${fmtBytes(du.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${execPct}%"></div></div>
        ${advancing ? `<p class="muted small">Estimated time remaining: ${escapeHtml(eta)}</p>` : ""}
        <p class="muted small">Beacon — ${fmtBytes(du.BeaconBytes)} of ~${fmtBytes(du.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${beaconPct}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${fmtBytes(du.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${escapeHtml(du.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${escapeHtml(du.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `;
  }

  // endpointsCard shows the local RPC URLs, live reachability dots (probed
  // on-box), and — for SSH targets — a copyable tunnel command plus the
  // spec's "local to the server" sentence.
  function endpointsCard(): string {
    if (endpointsErr) {
      return `
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${escapeHtml(endpointsErr)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;
    }
    if (!endpoints) {
      return `<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>`;
    }
    const ep = endpoints;
    const chainWarn =
      ep.ExecReachable && !ep.ChainIDMatches
        ? `<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`
        : "";
    const sshBlock =
      ep.Access === "ssh"
        ? `
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${escapeHtml(ep.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(ep.TunnelHint)}">Copy</button>
          </div>
        `
        : "";
    return `
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${dot(ep.ExecReachable ? "ok" : "bad")}
          <code class="endpoint-url">${escapeHtml(ep.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(ep.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${dot(ep.BeaconReachable ? "ok" : "bad")}
          <code class="endpoint-url">${escapeHtml(ep.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(ep.BeaconHTTP)}">Copy</button>
        </div>
        ${chainWarn}
        ${sshBlock}
      </div>
    `;
  }

  function serviceRow(svc: api.ServiceID, active: boolean): string {
    const label = SERVICE_LABEL[svc];
    const busy = pending[svc];
    const btn = (kind: api.ServiceActionKind, text: string, disabledWhen: boolean): string => {
      const isBusy = busy === kind;
      const disabled = busy !== null || disabledWhen;
      return `<button class="btn btn-ghost" data-action="svc-action" data-svc="${svc}" data-kind="${kind}" ${disabled ? "disabled" : ""}>${isBusy ? spinner() : escapeHtml(text)}</button>`;
    };
    return `
      <div class="service-row">
        <span>${escapeHtml(label)} ${active ? badge("active", "ok") : badge("down", "bad")}</span>
        <div class="service-actions">
          ${btn("start", "Start", active)}
          ${btn("stop", "Stop", !active)}
          ${btn("restart", "Restart", false)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${svc}" ${busy !== null ? "disabled" : ""}>Clear…</button>
        </div>
      </div>
    `;
  }

  function servicesCard(snap: api.Snapshot): string {
    return `
      <div class="card">
        <h3>Services</h3>
        ${serviceRow("exec", snap.execActive)}
        ${serviceRow("beacon", snap.beaconActive)}
        ${actionErr ? `<p class="error small">${escapeHtml(actionErr)}</p>` : ""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(targetId)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(targetId)}">Security →</a>
        </p>
      </div>
    `;
  }

  function spinner(): string {
    return `<span class="spinner" aria-label="working"></span>`;
  }

  async function runServiceAction(svc: api.ServiceID, kind: api.ServiceActionKind): Promise<void> {
    if (pending[svc] !== null) return;
    pending[svc] = kind;
    actionErr = null;
    render();
    try {
      await api.serviceAction(targetId, svc, kind);
    } catch (err) {
      actionErr = `${SERVICE_LABEL[svc]} ${kind} failed: ${err instanceof Error ? err.message : String(err)}`;
    }
    pending[svc] = null;
    if (!disposed) render();
  }

  async function copyButton(el: HTMLElement, text: string): Promise<void> {
    const ok = await copyToClipboard(text);
    const original = el.textContent;
    el.textContent = ok ? "Copied!" : "Copy failed";
    setTimeout(() => {
      if (!disposed) el.textContent = original;
    }, 1500);
  }

  // --- clear modal -------------------------------------------------------
  //
  // SIMPLIFICATION: the API doesn't expose the exact filesystem path(s) a
  // clear deletes (ops.ClearService derives them server-side from
  // catalog.Client.DataSubdirs, which isn't part of any response). Per the
  // brief's guidance, this modal shows the generic "<service> chain data
  // under the node's data directory" description plus the current size from
  // /du instead of the literal path(s).

  function openClearModal(svc: api.ServiceID): void {
    const label = SERVICE_LABEL[svc];
    const size = du ? fmtBytes(svc === "exec" ? du.ExecBytes : du.BeaconBytes) : "unknown (disk usage hasn't loaded)";

    openModal(
      `
        <h2>Clear ${escapeHtml(label)} data</h2>
        <p class="error">
          This stops the ${escapeHtml(label.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${escapeHtml(size)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${escapeHtml(svc)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action === "confirm") {
          void runClear(svc);
        }
      },
    );

    const input = document.getElementById("clear-confirm-input") as HTMLInputElement | null;
    const confirmBtn = document.getElementById("clear-confirm-btn") as HTMLButtonElement | null;
    input?.addEventListener("input", () => {
      if (confirmBtn) confirmBtn.disabled = input.value.trim() !== svc;
    });
    input?.focus();
  }

  async function runClear(svc: api.ServiceID): Promise<void> {
    const confirmBtn = document.getElementById("clear-confirm-btn") as HTMLButtonElement | null;
    if (confirmBtn) {
      confirmBtn.disabled = true;
      confirmBtn.textContent = "Clearing…";
    }
    try {
      await api.clearService(targetId, svc);
      closeModal();
      void loadDiskUsage();
    } catch (err) {
      const modalBody = document.querySelector<HTMLElement>("#clear-modal .modal");
      if (modalBody) {
        const msg = document.createElement("p");
        msg.className = "error small";
        msg.textContent = `Clear failed: ${err instanceof Error ? err.message : String(err)}`;
        modalBody.appendChild(msg);
      }
      if (confirmBtn) {
        confirmBtn.disabled = false;
        confirmBtn.textContent = "Clear and resync";
      }
    }
  }

  function openModal(innerHtml: string, onModalAction: (action: string) => void): void {
    closeModal();
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    overlay.id = "clear-modal";
    overlay.innerHTML = `<div class="modal">${innerHtml}</div>`;
    overlay.addEventListener("click", (ev) => {
      const t = (ev.target as HTMLElement).closest<HTMLElement>("[data-modal-action]");
      if (t?.dataset.modalAction) onModalAction(t.dataset.modalAction);
      if (ev.target === overlay) onModalAction("cancel");
    });
    document.body.appendChild(overlay);
  }

  function closeModal(): void {
    document.getElementById("clear-modal")?.remove();
  }

  return () => {
    disposed = true;
    streamStop?.();
    closeModal();
  };
}
