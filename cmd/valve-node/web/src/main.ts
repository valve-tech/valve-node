// Hash-routed SPA shell: #/targets, #/setup/<id>, #/dash/<id>, #/logs/<id>,
// #/settings. No framework — each screen module renders into a shared
// content element and returns a cleanup function (closes any EventSource /
// timers) that main.ts calls before routing away.
import "./style.css";
import { renderDashboard } from "./dashboard";
import { renderLogs } from "./logs";
import { renderSecurity } from "./security";
import { renderSettings } from "./settings";
import { renderShell } from "./ui";
import { renderTargets } from "./targets";
import { renderWizard } from "./wizard";

type Cleanup = () => void;

const appRoot = document.querySelector<HTMLDivElement>("#app")!;
const { contentEl, setActiveNav } = renderShell(appRoot);

let currentCleanup: Cleanup | null = null;

interface Route {
  screen: string;
  id?: string;
}

function parseHash(): Route {
  const hash = location.hash.replace(/^#\/?/, "");
  const parts = hash.split("/").filter(Boolean);
  if (parts.length === 0) return { screen: "targets" };
  const [screen, rawId] = parts;
  if (screen === "setup" || screen === "dash" || screen === "logs" || screen === "security") {
    return { screen, id: rawId ? decodeURIComponent(rawId) : undefined };
  }
  return { screen: screen ?? "targets" };
}

// mount gives a screen a brand-new child element of contentEl to render
// into and discards the previous one. Screens attach their delegated click
// listeners (via onAction) to the root element they're passed, not to
// contentEl itself — so a fresh node per screen means those listeners are
// discarded with the old node on every navigation instead of stacking up
// on the page-lifetime contentEl.
function mount(render: (root: HTMLElement) => Cleanup): Cleanup {
  const screenEl = document.createElement("div");
  contentEl.replaceChildren(screenEl);
  return render(screenEl);
}

function route(): void {
  if (currentCleanup) {
    try {
      currentCleanup();
    } catch {
      // A screen's cleanup failing must not block navigating away from it.
    }
    currentCleanup = null;
  }

  const { screen, id } = parseHash();
  setActiveNav(screen);

  switch (screen) {
    case "setup":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderWizard(root, id));
      break;
    case "dash":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderDashboard(root, id));
      break;
    case "logs":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderLogs(root, id));
      break;
    case "security":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderSecurity(root, id));
      break;
    case "settings":
      currentCleanup = mount((root) => renderSettings(root));
      break;
    case "targets":
    default:
      currentCleanup = mount((root) => renderTargets(root));
      break;
  }
}

window.addEventListener("hashchange", route);
route();
