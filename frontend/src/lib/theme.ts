import { writable } from "svelte/store";

// The app follows the OS theme until the user picks one explicitly; the choice
// is remembered. Applied as data-theme on the root element, which style.css
// overrides in both directions.
export type Theme = "light" | "dark";

const KEY = "gifly.theme";

function stored(): Theme | null {
  try {
    const s = localStorage.getItem(KEY);
    return s === "light" || s === "dark" ? s : null;
  } catch {
    return null;
  }
}

function osTheme(): Theme {
  try {
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  } catch {
    return "dark";
  }
}

export const theme = writable<Theme>(stored() ?? osTheme());

function apply(t: Theme) {
  document.documentElement.setAttribute("data-theme", t);
  try {
    localStorage.setItem(KEY, t);
  } catch {
    // ignore unwritable storage
  }
}

// initTheme applies the current theme to the document. Call once at startup.
export function initTheme(): void {
  let current: Theme = "dark";
  theme.subscribe((t) => (current = t))();
  apply(current);
}

export function toggleTheme(): void {
  theme.update((t) => {
    const next: Theme = t === "dark" ? "light" : "dark";
    apply(next);
    return next;
  });
}
