// #/settings — AI provider selection + key, plus the reference RPC base
// (advanced, rarely touched).
import * as api from "./api";
import { escapeHtml, footer, onAction } from "./ui";

const PROVIDERS: { value: api.AIProvider; label: string }[] = [
  { value: "", label: "None" },
  { value: "gemini", label: "Gemini" },
  { value: "groq", label: "Groq" },
  { value: "ollama", label: "Ollama" },
];

export function renderSettings(root: HTMLElement): () => void {
  let disposed = false;
  let keyTouched = false;
  let saving = false;
  let error: string | null = null;
  let saved = false;
  let current: api.Settings | null = null;

  root.innerHTML = `<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${footer()}`;
  const body = root.querySelector<HTMLElement>("#settings-body")!;

  onAction(root, (action) => {
    if (action === "save") void save();
    if (action === "clear-key") {
      if (!current) return;
      keyTouched = true;
      const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
      if (keyInput) keyInput.value = "";
      render(current);
    }
  });

  load();

  async function load(): Promise<void> {
    try {
      const settings = await api.getSettings();
      if (disposed) return;
      current = settings;
      render(settings);
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load settings: ${escapeHtml(String(err))}</p>`;
    }
  }

  function render(settings: api.Settings): void {
    const options = PROVIDERS.map(
      (p) => `<option value="${p.value}" ${settings.aiProvider === p.value ? "selected" : ""}>${escapeHtml(p.label)}</option>`,
    ).join("");

    body.innerHTML = `
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${options}</select>
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${settings.aiKeySet ? "•••••••• (leave blank to keep)" : "no key set"}" autocomplete="off" />
        </label>
        ${settings.aiKeySet ? `<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>` : ""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${escapeHtml(settings.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${error ? `<p class="error">${escapeHtml(error)}</p>` : ""}
        ${saved ? `<p class="ok">Saved.</p>` : ""}
        <button class="btn btn-primary" type="button" data-action="save" ${saving ? "disabled" : ""}>${saving ? "Saving…" : "Save"}</button>
      </form>
    `;

    const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
    keyInput?.addEventListener("input", () => {
      keyTouched = true;
      saved = false;
    });
    root.querySelector<HTMLSelectElement>("#ai-provider")?.addEventListener("change", () => {
      saved = false;
    });
    root.querySelector<HTMLInputElement>("#ref-rpc-base")?.addEventListener("input", () => {
      saved = false;
    });
  }

  async function save(): Promise<void> {
    const providerSel = root.querySelector<HTMLSelectElement>("#ai-provider");
    const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
    const refRpcInput = root.querySelector<HTMLInputElement>("#ref-rpc-base");
    if (!providerSel || !keyInput || !refRpcInput || !current) return;

    const body: api.PutSettingsRequest = {
      aiProvider: providerSel.value as api.AIProvider,
      refRpcBase: refRpcInput.value.trim(),
    };
    // Only send aiKey if the user actually touched the field this session —
    // omitting it preserves whatever key is already stored server-side.
    if (keyTouched) {
      body.aiKey = keyInput.value;
    }

    saving = true;
    error = null;
    saved = false;
    render(current);
    try {
      const updated = await api.putSettings(body);
      if (disposed) return;
      current = updated;
      keyTouched = false;
      saving = false;
      saved = true;
      render(updated);
    } catch (err) {
      if (disposed) return;
      saving = false;
      error = String(err instanceof Error ? err.message : err);
      render(current);
    }
  }

  return () => {
    disposed = true;
  };
}
