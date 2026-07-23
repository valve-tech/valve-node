// Small shared DOM/render helpers used by every screen. No framework — each
// screen renders a template string into a container and wires up event
// listeners by delegation afterwards.

export const LEARN_ROOT = "https://learn.valve.city/rpc";

// escapeHtml must wrap every piece of untrusted text (log lines, hostnames,
// target ids, server error messages, AI explanation text) before it's
// concatenated into an innerHTML template string.
export function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// footer renders the mandatory "learn how this works" link every screen
// carries, plus an optional per-context deep link (e.g. a specific
// network's or client's learn URL) alongside it.
export function footer(contextLabel?: string, contextUrl?: string): string {
  const context =
    contextLabel && contextUrl
      ? ` <span class="footer-sep">·</span> <a href="${escapeHtml(contextUrl)}" target="_blank" rel="noopener noreferrer">${escapeHtml(contextLabel)}</a>`
      : "";
  return `
    <footer class="footer">
      <a href="${escapeHtml(LEARN_ROOT)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${context}
    </footer>
  `;
}

export interface Shell {
  contentEl: HTMLElement;
  setActiveNav: (screen: string) => void;
}

// renderShell renders the app's persistent header/nav once and returns the
// content element the router swaps screens into.
export function renderShell(root: HTMLElement): Shell {
  root.innerHTML = `
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/targets">valve-node</a>
        <nav class="nav">
          <a href="#/targets" data-nav="targets">Targets</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;
  const contentEl = root.querySelector<HTMLElement>("#content")!;
  const navLinks = Array.from(root.querySelectorAll<HTMLAnchorElement>("[data-nav]"));
  const setActiveNav = (screen: string) => {
    for (const a of navLinks) {
      a.classList.toggle("active", a.dataset.nav === screen);
    }
  };
  return { contentEl, setActiveNav };
}

// fmtInt formats a number with thousands separators for readability
// (block/slot numbers get large fast).
export function fmtInt(n: number): string {
  return Number.isFinite(n) ? n.toLocaleString("en-US") : "—";
}

export function fmtPct(n: number): string {
  return Number.isFinite(n) ? `${n.toFixed(1)}%` : "—";
}

// fmtDuration renders a duration given in seconds as a short human string
// ("~2h 14m", "~45s"). Returns "—" for non-finite or negative input.
export function fmtDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  if (seconds < 60) return `~${Math.round(seconds)}s`;
  const totalMinutes = Math.round(seconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours === 0) return `~${minutes}m`;
  if (hours < 48) return `~${hours}h ${minutes}m`;
  const days = Math.floor(hours / 24);
  const remHours = hours % 24;
  return `~${days}d ${remHours}h`;
}

// badge renders a small colored status pill.
export function badge(text: string, kind: "ok" | "bad" | "warn" | "neutral"): string {
  return `<span class="badge badge-${kind}">${escapeHtml(text)}</span>`;
}

// dot renders a small reachability indicator (green/red/gray circle) —
// used where a full badge pill would be too heavy (e.g. next to a copyable
// URL on the endpoints card).
export function dot(kind: "ok" | "bad" | "neutral"): string {
  return `<span class="dot dot-${kind}"></span>`;
}

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB"];

// fmtBytes renders a byte count as a human-readable size ("3.9 TB", "512 MB").
export function fmtBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return "—";
  if (n === 0) return "0 B";
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const digits = value < 10 ? 2 : value < 100 ? 1 : 0;
  return `${value.toFixed(digits)} ${BYTE_UNITS[unit]}`;
}

// copyToClipboard writes text to the clipboard, returning whether it
// succeeded (the Clipboard API can be unavailable — insecure context, denied
// permission — and callers should show a fallback message rather than throw).
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

// on wires a delegated click handler for elements matching `[data-action]`
// inside container, calling handler(action, target) once per click.
export function onAction(
  container: HTMLElement,
  handler: (action: string, target: HTMLElement, ev: MouseEvent) => void,
): void {
  container.addEventListener("click", (ev) => {
    const target = (ev.target as HTMLElement).closest<HTMLElement>("[data-action]");
    if (!target || !container.contains(target)) return;
    const action = target.dataset.action;
    if (!action) return;
    handler(action, target, ev);
  });
}
