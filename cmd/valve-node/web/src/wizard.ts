// #/setup/<id> — the guided setup wizard: pick a network, pick a valid
// exec/beacon client pair, pick full vs. archive + a data dir, review, then
// run with live SSE step progress.
//
// DEVIATION FROM THE BRIEF: the brief's review screen was specified to
// "show the exact units to be written", but there is no API that renders
// units ahead of time (catalog.RenderUnits is a server-internal helper
// setup.Plan calls; it's never exposed over HTTP, and setup.Plan itself
// only exists once a run actually starts). The review screen below shows
// the WireConfig summary that will be POSTed, plus a step list — the step
// list is a small client-side mirror of setup.Plan's fixed step sequence
// (preflight, toolchain, install-exec, install-beacon, wire, start,
// handshake; see internal/setup/steps.go), not something the API returns.
import * as api from "./api";
import { badge, escapeHtml, footer, onAction } from "./ui";

type WizardStep = "network" | "clients" | "mode" | "review" | "run";

// STEP_PLAN mirrors internal/setup/steps.go's Plan() — a fixed sequence
// regardless of which clients are chosen (only the titles vary slightly,
// which we don't try to reproduce; the real titles come from the SSE
// stream's stepId once the run starts).
const STEP_PLAN: { id: string; title: string }[] = [
  { id: "preflight", title: "Preflight checks" },
  { id: "toolchain", title: "Ensure git + build toolchains" },
  { id: "install-exec", title: "Install execution client" },
  { id: "install-beacon", title: "Install beacon client" },
  { id: "wire", title: "Write JWT secret and systemd units" },
  { id: "start", title: "Start execution and beacon services" },
  { id: "handshake", title: "Verify execution/beacon handshake" },
];

const NETWORK_ORDER = [369, 943, 1];
const NETWORK_BADGE: Record<number, string> = {
  369: "default",
  943: "practise here first",
};

interface State {
  targetId: string;
  step: WizardStep;
  catalog: api.Catalog | null;
  loadError: string | null;
  chainId: number | null;
  execId: string | null;
  beaconId: string | null;
  archive: boolean;
  dataDir: string;
  jwtPath: string;
  starting: boolean;
  startError: string | null;
  events: api.SetupEvent[];
  streamStop: (() => void) | null;
}

export function renderWizard(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  const state: State = {
    targetId,
    step: "network",
    catalog: null,
    loadError: null,
    chainId: 369,
    execId: null,
    beaconId: null,
    archive: false,
    dataDir: "",
    jwtPath: "",
    starting: false,
    startError: null,
    events: [],
    streamStop: null,
  };

  root.innerHTML = `<h1>Setup: ${escapeHtml(targetId)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#wizard-body")!;
  const footerEl = root.querySelector<HTMLElement>("#wizard-footer")!;

  onAction(root, (action, el) => {
    handleAction(action, el);
  });

  load();

  async function load(): Promise<void> {
    try {
      const [catalog, targets] = await Promise.all([api.getCatalog(), api.listTargets()]);
      if (disposed) return;
      state.catalog = catalog;
      const existing = targets.find((t) => t.id === targetId);
      if (existing?.wire) {
        state.chainId = existing.wire.ChainID;
        state.execId = existing.wire.ExecID;
        state.beaconId = existing.wire.BeaconID;
        state.archive = existing.wire.Archive;
      }
      render();
    } catch (err) {
      if (disposed) return;
      state.loadError = String(err instanceof Error ? err.message : err);
      render();
    }
  }

  function render(): void {
    if (state.loadError) {
      body.innerHTML = `<p class="error">Failed to load: ${escapeHtml(state.loadError)}</p>`;
      return;
    }
    if (!state.catalog) return;

    body.innerHTML = `
      ${wizardProgress(state.step)}
      ${renderStep()}
    `;
    updateFooter();
  }

  // updateFooter refreshes the "learn more" link's per-context deep link to
  // the currently selected network's LearnURL, once a network is chosen.
  function updateFooter(): void {
    const net = state.catalog?.networks.find((n) => n.ChainID === state.chainId);
    footerEl.innerHTML = net ? footer(net.Name, net.LearnURL) : footer();
  }

  function renderStep(): string {
    switch (state.step) {
      case "network":
        return renderNetworkStep();
      case "clients":
        return renderClientsStep();
      case "mode":
        return renderModeStep();
      case "review":
        return renderReviewStep();
      case "run":
        return renderRunStep();
    }
  }

  function renderNetworkStep(): string {
    const catalog = state.catalog!;
    const cards = NETWORK_ORDER.map((chainId) => {
      const net = catalog.networks.find((n) => n.ChainID === chainId);
      if (!net) return "";
      const selected = state.chainId === chainId;
      const tag = NETWORK_BADGE[chainId] ? badge(NETWORK_BADGE[chainId]!, chainId === 369 ? "ok" : "warn") : "";
      return `
        <button class="card card-selectable ${selected ? "selected" : ""}" data-action="pick-network" data-chain-id="${chainId}" type="button">
          <h3>${escapeHtml(net.Name)} <span class="muted">(chain ${chainId})</span></h3>
          ${tag}
          <p class="muted small">Checkpoint sync from ${escapeHtml(net.CheckpointURL)}</p>
        </button>
      `;
    }).join("");

    return `
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${cards}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${state.chainId === null ? "disabled" : ""}>Next: clients</button>
        </div>
      </section>
    `;
  }

  function renderClientsStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    if (!net) return `<p class="error">Unknown network.</p>`;

    if (state.execId === null || !net.ExecClients.includes(state.execId)) {
      state.execId = net.ExecClients[0] ?? null;
    }
    if (state.beaconId === null || !net.BeaconClients.includes(state.beaconId)) {
      state.beaconId = net.BeaconClients[0] ?? null;
    }

    const execOptions = net.ExecClients.map((id) => clientOption(id, catalog, state.execId)).join("");
    const beaconOptions = net.BeaconClients.map((id) => clientOption(id, catalog, state.beaconId)).join("");

    return `
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${escapeHtml(net.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${execOptions}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${beaconOptions}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `;
  }

  function clientOption(id: string, catalog: api.Catalog, selectedId: string | null): string {
    const client = catalog.clients.find((c) => c.id === id);
    const label = client ? `${client.id} (${client.toolchain})` : id;
    return `<option value="${escapeHtml(id)}" ${id === selectedId ? "selected" : ""}>${escapeHtml(label)}</option>`;
  }

  function renderModeStep(): string {
    const defaultDataDir = state.chainId !== null ? `/var/lib/valve-node/${state.chainId}` : "";
    return `
      <section>
        <h2>3. Choose sync mode</h2>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${!state.archive ? "checked" : ""} />
          Full — prune old state, smaller disk footprint
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${state.archive ? "checked" : ""} />
          Archive — keep full history, needs much more disk
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${escapeHtml(defaultDataDir)})</span>
            <input id="data-dir-input" type="text" placeholder="${escapeHtml(defaultDataDir)}" value="${escapeHtml(state.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${escapeHtml(defaultDataDir)}/jwt.hex" value="${escapeHtml(state.jwtPath)}" />
          </label>
        </details>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-clients">Back</button>
          <button class="btn" data-action="goto-review">Next: review</button>
        </div>
      </section>
    `;
  }

  function renderReviewStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    const dataDir = state.dataDir || `/var/lib/valve-node/${state.chainId}`;
    const jwtPath = state.jwtPath || `${dataDir}/jwt.hex`;

    const stepRows = STEP_PLAN.map((s) => `<li>${escapeHtml(s.title)}</li>`).join("");

    return `
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${escapeHtml(state.targetId)}</td></tr>
            <tr><th>Network</th><td>${escapeHtml(net?.Name ?? String(state.chainId))} (chain ${state.chainId})</td></tr>
            <tr><th>Execution client</th><td>${escapeHtml(state.execId ?? "")}</td></tr>
            <tr><th>Beacon client</th><td>${escapeHtml(state.beaconId ?? "")}</td></tr>
            <tr><th>Mode</th><td>${state.archive ? "Archive" : "Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${escapeHtml(dataDir)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${escapeHtml(jwtPath)}</code></td></tr>
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${stepRows}</ol>
        ${state.startError ? `<p class="error">${escapeHtml(state.startError)}</p>` : ""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${state.starting ? "disabled" : ""}>
            ${state.starting ? "Starting…" : "Start setup"}
          </button>
        </div>
      </section>
    `;
  }

  function renderRunStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    const learnUrl = net?.LearnURL;

    const doneIds = new Set(state.events.filter((e) => e.done).map((e) => e.stepId));
    const erroredIds = new Set(state.events.filter((e) => e.err).map((e) => e.stepId));
    const linesByStep = new Map<string, string[]>();
    for (const ev of state.events) {
      if (!ev.line) continue;
      const list = linesByStep.get(ev.stepId) ?? [];
      list.push(ev.line);
      linesByStep.set(ev.stepId, list);
    }

    const items = STEP_PLAN.map((s) => {
      const isDone = doneIds.has(s.id);
      const isError = erroredIds.has(s.id);
      const mark = isError ? badge("failed", "bad") : isDone ? badge("done", "ok") : badge("pending", "neutral");
      const lines = (linesByStep.get(s.id) ?? []).slice(-5);
      const errLine = state.events.find((e) => e.stepId === s.id && e.err)?.err;
      const handshakeNote =
        s.id === "handshake"
          ? `<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${learnUrl ? ` <a href="${escapeHtml(learnUrl)}" target="_blank" rel="noopener noreferrer">Learn more →</a>` : ""}</p>`
          : "";
      return `
        <li class="step-row ${isDone ? "step-done" : ""} ${isError ? "step-error" : ""}">
          <div class="step-head">${mark} <strong>${escapeHtml(s.title)}</strong></div>
          ${handshakeNote}
          ${lines.length ? `<pre class="step-log">${lines.map((l) => escapeHtml(l)).join("\n")}</pre>` : ""}
          ${errLine ? `<p class="error small">${escapeHtml(errLine)}</p>` : ""}
        </li>
      `;
    }).join("");

    const anyError = state.events.some((e) => e.err);
    const allDone =
      STEP_PLAN.every((s) => doneIds.has(s.id)) || state.events.some((e) => e.stepId === "handshake" && e.done);

    return `
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${items}</ol>
        ${
          allDone && !anyError
            ? `<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(state.targetId)}">Open the dashboard →</a></p>`
            : ""
        }
        ${anyError ? `<button class="btn" data-action="start-setup">Retry setup</button>` : ""}
      </section>
    `;
  }

  function handleAction(action: string, el: HTMLElement): void {
    switch (action) {
      case "pick-network":
        state.chainId = Number(el.dataset.chainId);
        state.execId = null;
        state.beaconId = null;
        render();
        break;
      case "goto-network":
        state.step = "network";
        render();
        break;
      case "goto-clients":
        if (state.chainId === null) return;
        state.step = "clients";
        render();
        break;
      case "goto-mode":
        readClientSelects();
        state.step = "mode";
        render();
        break;
      case "goto-review":
        readModeInputs();
        state.step = "review";
        render();
        break;
      case "start-setup":
        void startSetup();
        break;
    }
  }

  function readClientSelects(): void {
    const execSel = root.querySelector<HTMLSelectElement>("#exec-select");
    const beaconSel = root.querySelector<HTMLSelectElement>("#beacon-select");
    if (execSel) state.execId = execSel.value;
    if (beaconSel) state.beaconId = beaconSel.value;
  }

  function readModeInputs(): void {
    const radios = root.querySelectorAll<HTMLInputElement>('input[name="mode"]');
    for (const r of Array.from(radios)) {
      if (r.checked) state.archive = r.value === "archive";
    }
    const dataDirInput = root.querySelector<HTMLInputElement>("#data-dir-input");
    const jwtPathInput = root.querySelector<HTMLInputElement>("#jwt-path-input");
    if (dataDirInput) state.dataDir = dataDirInput.value.trim();
    if (jwtPathInput) state.jwtPath = jwtPathInput.value.trim();
  }

  async function startSetup(): Promise<void> {
    if (state.chainId === null || !state.execId || !state.beaconId) return;
    state.starting = true;
    state.startError = null;
    render();

    const wire: api.StartSetupRequest = {
      ChainID: state.chainId,
      ExecID: state.execId,
      BeaconID: state.beaconId,
      Archive: state.archive,
    };
    if (state.dataDir) wire.DataDir = state.dataDir;
    if (state.jwtPath) wire.JWTPath = state.jwtPath;

    try {
      await api.startSetup(state.targetId, wire);
    } catch (err) {
      // 409 means a run is already in flight for this target — that's fine,
      // we just attach to its live stream below instead of starting a new
      // one.
      if (!(err instanceof api.ApiError && err.status === 409)) {
        state.starting = false;
        state.startError = String(err instanceof Error ? err.message : err);
        render();
        return;
      }
    }

    state.starting = false;
    state.step = "run";
    state.events = [];
    render();
    state.streamStop?.();
    state.streamStop = api.streamSetup(state.targetId, (ev) => {
      if (disposed) return;
      state.events.push(ev);
      if (state.step === "run") render();
    });
  }

  function wizardProgress(current: WizardStep): string {
    const steps: { id: WizardStep; label: string }[] = [
      { id: "network", label: "Network" },
      { id: "clients", label: "Clients" },
      { id: "mode", label: "Mode" },
      { id: "review", label: "Review" },
      { id: "run", label: "Run" },
    ];
    const order = steps.map((s) => s.id);
    const currentIdx = order.indexOf(current);
    return `
      <ol class="wizard-progress">
        ${steps
          .map((s, i) => {
            const cls = i === currentIdx ? "current" : i < currentIdx ? "past" : "future";
            return `<li class="${cls}">${escapeHtml(s.label)}</li>`;
          })
          .join("")}
      </ol>
    `;
  }

  return () => {
    disposed = true;
    state.streamStop?.();
  };
}
