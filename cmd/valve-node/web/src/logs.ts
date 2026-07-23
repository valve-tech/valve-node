// #/logs/<id> — live log tail with severity coloring, an error feed panel,
// and an "Explain with AI" modal.
import * as api from "./api";
import { badge, escapeHtml, footer, onAction } from "./ui";

// maxRenderedLines caps how many tail rows stay in the DOM — the server's
// own ring buffer already caps history (internal/logwatch: 1000), this just
// keeps the live view from growing unbounded over a long session.
const maxRenderedLines = 500;

// maxExplainLines mirrors handleExplain's own default cap
// (maxDefaultExplainHits in internal/server/api.go) so the consent modal
// shows exactly what would be auto-selected server-side, capped the same
// way.
const maxExplainLines = 40;

const CONSENT_KEY = "valve-node.explain-consent";

type Severity = "info" | "warn" | "error" | "critical";

export function renderLogs(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let streamStop: (() => void) | null = null;
  const hits: api.Hit[] = [];

  root.innerHTML = `
    <h1>Logs: ${escapeHtml(targetId)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    ${footer()}
  `;
  const body = root.querySelector<HTMLElement>("#logs-body")!;

  onAction(root, (action) => {
    if (action === "explain") void openExplainFlow();
  });

  init();

  async function init(): Promise<void> {
    let target: api.Target | undefined;
    try {
      const targets = await api.listTargets();
      target = targets.find((t) => t.id === targetId);
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

    try {
      const recent = await api.getLogs(targetId, 200);
      if (disposed) return;
      hits.push(...recent);
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load logs: ${escapeHtml(String(err))}</p>`;
      return;
    }

    renderAll();
    streamStop = api.streamLogs(targetId, (hit) => {
      if (disposed) return;
      hits.push(hit);
      if (hits.length > maxRenderedLines) hits.splice(0, hits.length - maxRenderedLines);
      renderAll();
    });
  }

  function renderAll(): void {
    const errorHits = hits.filter((h) => h.severity === "error" || h.severity === "critical");
    body.innerHTML = `
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${hits.map(logLine).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${badge(String(errorHits.length), errorHits.length ? "bad" : "neutral")}</h2>
          <div class="log-lines">${
            errorHits.length ? errorHits.slice().reverse().map(logLine).join("") : `<p class="muted">No errors seen yet.</p>`
          }</div>
        </section>
      </div>
    `;
    const tail = body.querySelector<HTMLElement>(".log-lines");
    if (tail) tail.scrollTop = tail.scrollHeight;
  }

  function logLine(h: api.Hit): string {
    const sev = (h.severity as Severity) || "info";
    const learn = h.learnUrl
      ? ` <a href="${escapeHtml(h.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`
      : "";
    return `
      <div class="log-line log-${escapeHtml(sev)}">
        <span class="log-time">${escapeHtml(new Date(h.at).toLocaleTimeString())}</span>
        <span class="log-unit">${escapeHtml(h.unit)}</span>
        <span class="log-sev">${escapeHtml(sev)}</span>
        <span class="log-text">${escapeHtml(h.line)}</span>
        ${h.explain ? `<div class="log-explain">${escapeHtml(h.explain)}${learn}</div>` : ""}
      </div>
    `;
  }

  // --- Explain with AI -------------------------------------------------

  async function openExplainFlow(): Promise<void> {
    const errorLines = hits
      .filter((h) => h.severity === "error" || h.severity === "critical")
      .map((h) => h.line)
      .slice(-maxExplainLines);

    const consented = localStorage.getItem(CONSENT_KEY) === "1";
    if (!consented) {
      showConsentModal(errorLines);
      return;
    }
    await runExplain(errorLines);
  }

  function showConsentModal(candidateLines: string[]): void {
    const excerptHtml = candidateLines.length
      ? `<pre class="explain-excerpt">${candidateLines.map((l) => escapeHtml(l)).join("\n")}</pre>`
      : `<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>`;

    openModal(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${excerptHtml}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `, (action) => {
      if (action === "proceed") {
        localStorage.setItem(CONSENT_KEY, "1");
        closeModal();
        void runExplain(candidateLines);
      } else {
        closeModal();
      }
    });
  }

  async function runExplain(lines: string[]): Promise<void> {
    openModal(`<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>`, () => {});
    try {
      const res = lines.length ? await api.explain(targetId, lines) : await api.explain(targetId);
      if (disposed) return;
      openModal(`
        <h2>Explanation</h2>
        <div class="explain-text">${escapeHtml(res.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${res.sentExcerpt.map((l) => escapeHtml(l)).join("\n") || "(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `, (action) => {
        if (action === "close") closeModal();
      });
    } catch (err) {
      if (disposed) return;
      if (err instanceof api.ApiError && err.status === 409) {
        openModal(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `, (action) => {
          if (action === "close") closeModal();
        });
        return;
      }
      openModal(`
        <h2>Explain failed</h2>
        <p class="error">${escapeHtml(err instanceof Error ? err.message : String(err))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `, (action) => {
        if (action === "close") closeModal();
      });
    }
  }

  function openModal(innerHtml: string, onModalAction: (action: string) => void): void {
    closeModal();
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    overlay.id = "explain-modal";
    overlay.innerHTML = `<div class="modal">${innerHtml}</div>`;
    overlay.addEventListener("click", (ev) => {
      const t = (ev.target as HTMLElement).closest<HTMLElement>("[data-modal-action]");
      if (t?.dataset.modalAction) onModalAction(t.dataset.modalAction);
      if (ev.target === overlay) onModalAction("cancel");
    });
    document.body.appendChild(overlay);
  }

  function closeModal(): void {
    document.getElementById("explain-modal")?.remove();
  }

  return () => {
    disposed = true;
    streamStop?.();
    closeModal();
  };
}
