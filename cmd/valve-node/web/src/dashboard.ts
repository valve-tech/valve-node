// #/dash/<id> — live sync/peer/disk dashboard, fed by the target's 5s
// monitor SSE stream (see internal/monitor: the poller ticks every 5s
// server-side; this file just renders whatever it sends).
import * as api from "./api";
import { badge, escapeHtml, fmtDuration, fmtInt, fmtPct, footer } from "./ui";

// highDiskUsagePct is the threshold above which the disk card switches from
// its normal styling to a warning one.
const highDiskUsagePct = 85;

export function renderDashboard(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let streamStop: (() => void) | null = null;
  let prevSnapshot: api.Snapshot | null = null;
  let execBlocksPerSec: number | null = null;

  root.innerHTML = `<h1>Dashboard: ${escapeHtml(targetId)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#dash-body")!;
  const footerEl = root.querySelector<HTMLElement>("#dash-footer")!;

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
      renderSnapshot(snap);
      prevSnapshot = snap;
    });
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

  function renderSnapshot(snap: api.Snapshot): void {
    body.innerHTML = `
      <div class="card-grid">
        ${execSyncCard(snap)}
        ${beaconSyncCard(snap)}
        ${peersCard(snap)}
        ${diskCard(snap)}
        ${servicesCard(snap)}
      </div>
      <p class="muted small">Last updated ${escapeHtml(new Date(snap.at).toLocaleTimeString())}</p>
    `;
  }

  function execSyncCard(snap: api.Snapshot): string {
    const hasRef = snap.refHead > 0;
    const lag = hasRef ? snap.refHead - snap.execHead : null;
    const eta =
      lag !== null && lag > 0 && execBlocksPerSec && execBlocksPerSec > 0 ? fmtDuration(lag / execBlocksPerSec) : lag !== null && lag <= 0 ? "caught up" : "—";

    return `
      <div class="card">
        <h3>Execution sync</h3>
        <p>${snap.execSyncing ? badge("syncing", "warn") : badge("synced", "ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${fmtInt(snap.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${hasRef ? fmtInt(snap.refHead) : "unavailable"}</dd></div>
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

  function servicesCard(snap: api.Snapshot): string {
    return `
      <div class="card">
        <h3>Services</h3>
        <p>Execution ${snap.execActive ? badge("active", "ok") : badge("down", "bad")}</p>
        <p>Beacon ${snap.beaconActive ? badge("active", "ok") : badge("down", "bad")}</p>
        <p><a href="#/logs/${encodeURIComponent(targetId)}">View logs →</a></p>
      </div>
    `;
  }

  return () => {
    disposed = true;
    streamStop?.();
  };
}
