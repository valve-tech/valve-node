const app = document.querySelector<HTMLDivElement>("#app")!;

app.innerHTML = `
  <h1>valve-node</h1>
  <p id="status">checking API…</p>
`;

const statusEl = document.querySelector<HTMLParagraphElement>("#status")!;

fetch("/api/health")
  .then((res) => res.json())
  .then((data: { ok: boolean }) => {
    statusEl.textContent = data.ok ? "API OK" : "API not OK";
  })
  .catch((err: unknown) => {
    statusEl.textContent = `API error: ${String(err)}`;
  });
