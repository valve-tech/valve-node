// #/security/<id> — the firewall/exposure checklist from internal/ops's
// FirewallChecklist: status chips, expandable why/fix per item (with a copy
// button on the fix command), and a "Re-run checks" button that re-probes
// live. Every probe here is read-only on the server side — this screen never
// sends a mutating command, only ever copies a suggested one to the
// clipboard for the operator to review and run themselves.
import * as api from "./api";
import { badge, copyToClipboard, escapeHtml, footer, onAction } from "./ui";

export function renderSecurity(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let items: api.CheckItem[] = [];
  let loadErr: string | null = null;
  let loading = false;
  let loaded = false;

  root.innerHTML = `<h1>Security: ${escapeHtml(targetId)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#sec-body")!;
  const footerEl = root.querySelector<HTMLElement>("#sec-footer")!;

  onAction(root, (action, el) => {
    if (action === "rerun") {
      void load();
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

    await load();
  }

  async function load(): Promise<void> {
    loading = true;
    loadErr = null;
    render();
    try {
      items = await api.getFirewallChecklist(targetId);
      loaded = true;
    } catch (err) {
      loadErr = String(err instanceof Error ? err.message : err);
    }
    loading = false;
    if (!disposed) render();
  }

  function render(): void {
    body.innerHTML = `
      <p><a href="#/dash/${encodeURIComponent(targetId)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${loading ? "disabled" : ""}>${loading ? "Re-running…" : "Re-run checks"}</button>
      </div>
      ${loadErr ? `<p class="error">${escapeHtml(loadErr)}</p>` : ""}
      ${
        !loaded && loading
          ? `<p class="muted">Loading…</p>`
          : items.length
            ? `<ul class="check-list">${items.map(checkItemHtml).join("")}</ul>`
            : loaded
              ? `<p class="muted">No checks returned.</p>`
              : ""
      }
    `;
  }

  function checkItemHtml(item: api.CheckItem): string {
    const kind = item.Status === "pass" ? "ok" : item.Status === "fail" ? "bad" : item.Status === "warn" ? "warn" : "neutral";
    return `
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${badge(item.Status, kind)}
          <strong>${escapeHtml(item.Title)}</strong>
          <span class="muted small check-detail-inline">${escapeHtml(item.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${escapeHtml(item.Why)}</p>
          </details>
          ${
            item.Fix
              ? `
                <details open>
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
