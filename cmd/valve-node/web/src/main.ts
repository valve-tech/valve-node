// Hash-routed SPA shell: #/targets, #/setup/<id>, #/dash/<id>, #/logs/<id>,
// #/settings. No framework — each screen module renders into a shared
// content element and returns a cleanup function (closes any EventSource /
// timers) that main.ts calls before routing away.
import "./style.css";
import { renderDashboard } from "./dashboard";
import { renderLogs } from "./logs";
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
  if (screen === "setup" || screen === "dash" || screen === "logs") {
    return { screen, id: rawId ? decodeURIComponent(rawId) : undefined };
  }
  return { screen: screen ?? "targets" };
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
      currentCleanup = renderWizard(contentEl, id);
      break;
    case "dash":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = renderDashboard(contentEl, id);
      break;
    case "logs":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = renderLogs(contentEl, id);
      break;
    case "settings":
      currentCleanup = renderSettings(contentEl);
      break;
    case "targets":
    default:
      currentCleanup = renderTargets(contentEl);
      break;
  }
}

window.addEventListener("hashchange", route);
route();
