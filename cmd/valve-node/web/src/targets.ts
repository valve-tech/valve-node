// #/targets — the local-machine card, the list of SSH targets, and the
// "add server over SSH" form.
import * as api from "./api";
import { badge, escapeHtml, footer, onAction } from "./ui";

const LOCAL_TARGET_ID = "local";

export function renderTargets(root: HTMLElement): () => void {
  let disposed = false;

  root.innerHTML = `
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${footer()}
  `;
  const body = root.querySelector<HTMLElement>("#targets-body")!;

  onAction(root, (action, el) => {
    void handleAction(action, el);
  });

  load();

  async function load(): Promise<void> {
    try {
      const [targets, catalog] = await Promise.all([api.listTargets(), api.getCatalog()]);
      if (disposed) return;
      renderBody(targets, catalog);
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load targets: ${escapeHtml(String(err))}</p>`;
    }
  }

  function renderBody(targets: api.Target[], catalog: api.Catalog): void {
    const local = targets.find((t) => t.mode === "local");
    const remote = targets.filter((t) => t.mode === "ssh");

    const localSection = local
      ? targetCard(local, catalog)
      : `
        <div class="card">
          <h2>This machine</h2>
          ${localNudge()}
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `;

    const remoteCards = remote.length
      ? remote.map((t) => targetCard(t, catalog)).join("")
      : `<p class="muted">No SSH targets yet.</p>`;

    body.innerHTML = `
      <section class="section">${localSection}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${remoteCards}</div>
        ${sshFormMarkup()}
      </section>
    `;
  }

  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    if (action === "add-local") {
      await addLocal();
      return;
    }
    if (action === "delete-target") {
      const id = el.dataset.id;
      if (!id) return;
      if (!confirm(`Remove target "${id}"? This does not touch anything already running on it.`)) {
        return;
      }
      await deleteTarget(id);
      return;
    }
    if (action === "add-ssh") {
      await addSSH();
    }
  }

  async function addLocal(): Promise<void> {
    clearFormError();
    try {
      await api.addTarget({ id: LOCAL_TARGET_ID, mode: "local" });
      await load();
    } catch (err) {
      showFormError(err);
    }
  }

  async function deleteTarget(id: string): Promise<void> {
    try {
      await api.deleteTarget(id);
      await load();
    } catch (err) {
      showFormError(err);
    }
  }

  async function addSSH(): Promise<void> {
    const hostEl = root.querySelector<HTMLInputElement>("#ssh-host");
    const userEl = root.querySelector<HTMLInputElement>("#ssh-user");
    const keyEl = root.querySelector<HTMLInputElement>("#ssh-key");
    const portEl = root.querySelector<HTMLInputElement>("#ssh-port");
    const idEl = root.querySelector<HTMLInputElement>("#ssh-id");
    if (!hostEl || !userEl || !keyEl || !portEl || !idEl) return;

    const host = hostEl.value.trim();
    const user = userEl.value.trim();
    const keyPath = keyEl.value.trim();
    const portRaw = portEl.value.trim();
    const idRaw = idEl.value.trim();

    clearFormError();
    if (!host || !user || !keyPath) {
      showFormError(new Error("host, user, and key path are required"));
      return;
    }

    const id = idRaw || slugify(host);
    const ssh: api.SSHConfig = { Host: host, User: user, KeyPath: keyPath };
    if (portRaw) {
      const port = Number.parseInt(portRaw, 10);
      if (!Number.isFinite(port) || port <= 0) {
        showFormError(new Error("port must be a positive number"));
        return;
      }
      ssh.Port = port;
    }

    const submitBtn = root.querySelector<HTMLButtonElement>("#ssh-submit");
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = "Connecting…";
    }
    try {
      await api.addTarget({ id, mode: "ssh", ssh });
      await load();
    } catch (err) {
      showFormError(err);
      if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = "Add server";
      }
    }
  }

  function showFormError(err: unknown): void {
    let box = root.querySelector<HTMLElement>("#targets-error");
    if (!box) {
      body.insertAdjacentHTML("afterbegin", `<p id="targets-error" class="error"></p>`);
      box = root.querySelector<HTMLElement>("#targets-error")!;
    }
    box.textContent = String(err instanceof Error ? err.message : err);
  }

  function clearFormError(): void {
    root.querySelector<HTMLElement>("#targets-error")?.remove();
  }

  return () => {
    disposed = true;
  };
}

function targetCard(t: api.Target, catalog: api.Catalog): string {
  const wire = t.wire;
  const modeLabel = t.mode === "local" ? "this machine" : "SSH";
  const location = t.mode === "ssh" && t.ssh ? `${escapeHtml(t.ssh.User)}@${escapeHtml(t.ssh.Host)}` : modeLabel;

  let statusLine: string;
  let actions: string;
  if (!wire) {
    statusLine = badge("not set up", "neutral");
    actions = `<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;
  } else {
    const net = catalog.networks.find((n) => n.ChainID === wire.ChainID);
    const netName = net ? net.Name : `chain ${wire.ChainID}`;
    statusLine = `${badge(netName, "ok")} ${badge(wire.ExecID, "neutral")} ${badge(wire.BeaconID, "neutral")}${wire.Archive ? " " + badge("archive", "warn") : ""}`;
    actions = `
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `;
  }

  return `
    <div class="card">
      <h2>${escapeHtml(t.id)}</h2>
      <p class="muted">${location}</p>
      <p>${statusLine}</p>
      <div class="card-actions">
        ${actions}
        <button class="btn btn-danger" data-action="delete-target" data-id="${escapeHtml(t.id)}">Remove</button>
      </div>
    </div>
  `;
}

function sshFormMarkup(): string {
  return `
    <form class="card" id="ssh-add-form" onsubmit="return false">
      <h3>Add server over SSH</h3>
      <label>
        Host
        <input id="ssh-host" type="text" placeholder="203.0.113.10" autocomplete="off" />
      </label>
      <label>
        User
        <input id="ssh-user" type="text" placeholder="root" autocomplete="off" />
      </label>
      <label>
        Private key path
        <input id="ssh-key" type="text" placeholder="/home/me/.ssh/id_ed25519" autocomplete="off" />
      </label>
      <label>
        Port <span class="muted">(optional, default 22)</span>
        <input id="ssh-port" type="text" inputmode="numeric" placeholder="22" autocomplete="off" />
      </label>
      <label>
        Target name <span class="muted">(optional, defaults to the host)</span>
        <input id="ssh-id" type="text" placeholder="my-node" autocomplete="off" />
      </label>
      <p class="muted small">
        The key never leaves this machine — only its path is stored, and the
        connection is dialed immediately so the host key can be pinned
        (trust-on-first-use) before it's saved.
      </p>
      <button class="btn" type="button" id="ssh-submit" data-action="add-ssh">Add server</button>
    </form>
  `;
}

// looksNonLinux is a best-effort guess at whether the browser is running on
// the same non-Linux machine as the valve-node binary — the API exposes no
// way to ask the server's own OS directly, so this reads the browser's
// platform as a proxy (accurate in the common case: binary and browser on
// the same desktop). See the Task 8 report for the caveat.
function looksNonLinux(): boolean {
  const uaData = (navigator as Navigator & { userAgentData?: { platform?: string } }).userAgentData;
  const platform = uaData?.platform || navigator.platform || navigator.userAgent;
  return /mac|win/i.test(platform) && !/linux|android/i.test(platform);
}

// localNudge renders the local card's guidance. On a machine that doesn't
// look like Linux this is a real warning banner (spec v0.2 §1's exact
// wording), not just a muted note — macOS/Windows setup fails preflight's
// Linux check anyway, so this makes that a first-class message instead of a
// surprise partway through the wizard.
function localNudge(): string {
  if (looksNonLinux()) {
    return `
      <p class="banner banner-warn">
        macOS and Windows are not supported node hosts — use this machine as a controller and add a
        Linux server over SSH.
      </p>
    `;
  }
  return `<p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>`;
}

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "") || "target";
}
