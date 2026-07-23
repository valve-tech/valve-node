// #/diag/<id> — the network diagnostics ladder from internal/ops's
// NetworkDiagnostics. Checks run in order and stop at the first failure, so
// the report reads "check, check, check — failed HERE". Runs happen two
// ways: automatically on the server when a journal error signature fires or
// a connection fails (inactive service, zero peers), and manually from the
// button here. This screen shows the latest report either way. Every probe
// is read-only on the server side — this screen never sends a mutating
// command, only ever copies a suggested fix to the clipboard for the
// operator to review and run themselves.
import * as api from "./api";
import { badge, copyToClipboard, escapeHtml, footer, onAction } from "./ui";

export function renderDiagnostics(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let report: api.DiagReport | null = null;
  let loadErr: string | null = null;
  let running = false;
  let loaded = false;

  root.innerHTML = `<h1>Network diagnostics: ${escapeHtml(targetId)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#diag-body")!;
  const footerEl = root.querySelector<HTMLElement>("#diag-footer")!;

  onAction(root, (action, el) => {
    if (action === "run") {
      void run();
    } else if (action === "toggle") {
      el.closest<HTMLElement>(".check-item")?.classList.toggle("expanded");
    } else if (action === "copy") {
      const text = el.dataset.copy;
      if (text) void copyButton(el, text);
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

    try {
      report = await api.getLatestDiagnostics(targetId);
      loaded = true;
    } catch (err) {
      loadErr = String(err instanceof Error ? err.message : err);
    }
    if (!disposed) render();
  }

  async function run(): Promise<void> {
    running = true;
    loadErr = null;
    render();
    try {
      report = await api.runNetworkDiagnostics(targetId);
      loaded = true;
    } catch (err) {
      loadErr = String(err instanceof Error ? err.message : err);
    }
    running = false;
    if (!disposed) render();
  }

  function render(): void {
    body.innerHTML = `
      <p><a href="#/dash/${encodeURIComponent(targetId)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${running ? "disabled" : ""}>${running ? "Running…" : "Run diagnostics"}</button>
      </div>
      ${loadErr ? `<p class="error">${escapeHtml(loadErr)}</p>` : ""}
      ${reportHtml()}
    `;
  }

  function reportHtml(): string {
    if (!loaded && !loadErr) return `<p class="muted">Loading…</p>`;
    if (!report) return `<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;

    const when = new Date(report.at).toLocaleString();
    const verdict = report.failedId
      ? `<p><strong>Failed at: ${escapeHtml(titleOf(report.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`
      : `<p><strong>All checks passed.</strong></p>`;
    return `
      <p class="muted small">Last run ${escapeHtml(when)} — trigger: ${escapeHtml(report.trigger)}</p>
      ${verdict}
      <ul class="check-list">${report.items.map(checkItemHtml).join("")}</ul>
    `;
  }

  function titleOf(id: string): string {
    return report?.items.find((it) => it.ID === id)?.Title ?? id;
  }

  function checkItemHtml(item: api.CheckItem): string {
    const kind = item.Status === "pass" ? "ok" : item.Status === "fail" ? "bad" : item.Status === "warn" ? "warn" : "neutral";
    const failedHere = item.ID === report?.failedId;
    return `
      <li class="check-item${failedHere ? " expanded" : ""}">
        <button class="check-head" data-action="toggle" type="button">
          ${badge(failedHere ? "failed here" : item.Status, kind)}
          <strong>${escapeHtml(item.Title)}</strong>
          <span class="muted small check-detail-inline">${escapeHtml(item.Detail)}</span>
        </button>
        <div class="check-body">
          <details${failedHere ? " open" : ""}>
            <summary>Why this matters</summary>
            <p class="muted small">${escapeHtml(item.Why)}</p>
          </details>
          ${
            item.Fix
              ? `
                <details${failedHere ? " open" : ""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${escapeHtml(item.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(item.Fix)}">Copy</button>
                </details>
              `
              : ""
          }
        </div>
      </li>
    `;
  }

  async function copyButton(el: HTMLElement, text: string): Promise<void> {
    const ok = await copyToClipboard(text);
    const original = el.textContent;
    el.textContent = ok ? "Copied!" : "Copy failed";
    setTimeout(() => {
      if (!disposed) el.textContent = original;
    }, 1500);
  }

  return () => {
    disposed = true;
  };
}
