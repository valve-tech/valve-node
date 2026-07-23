// Thin typed wrapper around valve-node's JSON/SSE API (internal/server/api.go).
// Every type here mirrors the *actual wire shape* of the corresponding Go
// struct, not a guessed camelCase version of it: several response structs
// (catalog.Network, executor.SSHConfig, catalog.WireConfig) carry no `json`
// tags, so encoding/json marshals them using the bare (capitalized) Go field
// name. Where a struct DOES carry explicit tags (catalogClient, setup.Event,
// monitor.Snapshot, logwatch.Hit, settings), this file uses the tagged
// lowerCamelCase names. Verified against the real server via a scratch
// marshal/unmarshal test, not just by reading the struct definitions.
//
// The browser sends the session cookie automatically on same-origin fetch
// and EventSource calls, so nothing here needs an Authorization header.

// ---------------------------------------------------------------------
// catalog
// ---------------------------------------------------------------------

export interface Network {
  ChainID: number;
  Name: string;
  CheckpointURL: string;
  ExecClients: string[];
  BeaconClients: string[];
  LearnURL: string;
}

export interface CatalogClient {
  id: string;
  kind: string;
  repo: string;
  pinVersion: string;
  toolchain: string;
  learnUrl: string;
}

export interface Catalog {
  networks: Network[];
  clients: CatalogClient[];
}

export function getCatalog(): Promise<Catalog> {
  return request<Catalog>("/api/catalog");
}

// ---------------------------------------------------------------------
// targets
// ---------------------------------------------------------------------

export interface SSHConfig {
  Host: string;
  User: string;
  KeyPath: string;
  HostKeyFile?: string;
  Port?: number;
}

export interface WireConfig {
  ChainID: number;
  ExecID: string;
  BeaconID: string;
  DataDir: string;
  JWTPath: string;
  Archive: boolean;
}

export type TargetMode = "local" | "ssh";

export interface Target {
  id: string;
  mode: TargetMode;
  ssh?: SSHConfig;
  wire?: WireConfig;
}

export function listTargets(): Promise<Target[]> {
  return request<Target[]>("/api/targets");
}

export interface AddTargetRequest {
  id: string;
  mode: TargetMode;
  ssh?: SSHConfig;
}

export function addTarget(body: AddTargetRequest): Promise<Target> {
  return request<Target>("/api/targets", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

export function deleteTarget(id: string): Promise<void> {
  return request<void>(`/api/targets/${encodeURIComponent(id)}`, { method: "DELETE" });
}

// ---------------------------------------------------------------------
// setup wizard
// ---------------------------------------------------------------------

// StartSetupRequest is catalog.WireConfig's request shape. DataDir/JWTPath
// are optional — the server fills in `/var/lib/valve-node/<chainId>` and
// `<dataDir>/jwt.hex` when omitted (see handleStartSetup).
export interface StartSetupRequest {
  ChainID: number;
  ExecID: string;
  BeaconID: string;
  DataDir?: string;
  JWTPath?: string;
  Archive: boolean;
}

export function startSetup(id: string, wire: StartSetupRequest): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/targets/${encodeURIComponent(id)}/setup`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(wire),
  });
}

export interface SetupEvent {
  stepId: string;
  line?: string;
  done?: boolean;
  err?: string;
}

export function streamSetup(id: string, onEvent: (ev: SetupEvent) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/setup/stream`);
  es.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data) as SetupEvent);
    } catch {
      // Malformed frame — drop it rather than crash the stream handler.
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// monitor
// ---------------------------------------------------------------------

export interface Snapshot {
  at: string;
  execSyncing: boolean;
  execHead: number;
  refHead: number;
  beaconSlot: number;
  beaconDistance: number;
  execPeers: number;
  beaconPeers: number;
  diskUsedPct: number;
  execActive: boolean;
  beaconActive: boolean;
}

export function streamMonitor(id: string, onSnapshot: (s: Snapshot) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/monitor/stream`);
  es.onmessage = (msg) => {
    try {
      onSnapshot(JSON.parse(msg.data) as Snapshot);
    } catch {
      // ignore malformed frame
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// logs
// ---------------------------------------------------------------------

export interface Hit {
  unit: string;
  line: string;
  at: string;
  signature: string;
  severity: string; // info|warn|error|critical
  explain: string;
  learnUrl?: string;
}

export function getLogs(id: string, n = 200): Promise<Hit[]> {
  return request<Hit[]>(`/api/targets/${encodeURIComponent(id)}/logs?n=${n}`);
}

export function streamLogs(id: string, onHit: (h: Hit) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/logs/stream`);
  es.onmessage = (msg) => {
    try {
      onHit(JSON.parse(msg.data) as Hit);
    } catch {
      // ignore malformed frame
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// explain
// ---------------------------------------------------------------------

export interface ExplainResponse {
  text: string;
  sentExcerpt: string[];
}

// explain, when `lines` is omitted (undefined), lets the server auto-select
// the target's recent error/critical log lines. Pass an explicit array
// (even []) to control exactly what gets sent — logs.ts uses this so the
// consent modal can show the operator the exact excerpt before it goes out.
export function explain(id: string, lines?: string[]): Promise<ExplainResponse> {
  const body = lines === undefined ? {} : { lines };
  return request<ExplainResponse>(`/api/targets/${encodeURIComponent(id)}/explain`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

// ---------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------

export type AIProvider = "" | "gemini" | "groq" | "ollama";

export interface Settings {
  aiProvider: AIProvider;
  aiKeySet: boolean;
  refRpcBase: string;
}

export function getSettings(): Promise<Settings> {
  return request<Settings>("/api/settings");
}

export interface PutSettingsRequest {
  aiProvider?: AIProvider;
  // aiKey omitted => leave the stored key unchanged. "" => explicitly clear
  // it. Never send a field the user hasn't touched.
  aiKey?: string;
  refRpcBase?: string;
}

export function putSettings(body: PutSettingsRequest): Promise<Settings> {
  return request<Settings>("/api/settings", {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

// ---------------------------------------------------------------------
// fetch plumbing
// ---------------------------------------------------------------------

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const JSON_HEADERS = { "Content-Type": "application/json" };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = res.statusText || `HTTP ${res.status}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body && typeof body.error === "string" && body.error) {
        message = body.error;
      }
    } catch {
      // body wasn't JSON (or was empty) — fall back to statusText.
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : (undefined as T);
}
