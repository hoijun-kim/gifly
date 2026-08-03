import { settings, type Settings, type FolderMode, type OnExist } from "./settings";

// Only the save-destination preferences are remembered across restarts - the
// per-source name and the creative options are not sticky.
type Persisted = Pick<Settings, "outFolderMode" | "outFolder" | "onExist" | "autoReveal">;

const KEY = "gifly.save.v1";
const FOLDER_MODES: FolderMode[] = ["source", "custom"];
const ON_EXISTS: OnExist[] = ["overwrite", "number", "timestamp"];

// clean keeps only well-formed values from a possibly-corrupt localStorage blob.
function clean(p: unknown): Partial<Persisted> {
  if (!p || typeof p !== "object") return {};
  const o = p as Record<string, unknown>;
  const out: Partial<Persisted> = {};
  if (typeof o.outFolderMode === "string" && FOLDER_MODES.includes(o.outFolderMode as FolderMode)) {
    out.outFolderMode = o.outFolderMode as FolderMode;
  }
  if (typeof o.outFolder === "string") out.outFolder = o.outFolder;
  if (typeof o.onExist === "string" && ON_EXISTS.includes(o.onExist as OnExist)) {
    out.onExist = o.onExist as OnExist;
  }
  if (typeof o.autoReveal === "boolean") out.autoReveal = o.autoReveal;
  return out;
}

// initPersistence loads the remembered save preferences, then keeps them in
// sync with localStorage. Safe to call once at startup; a missing or blocked
// localStorage is tolerated.
export function initPersistence(): void {
  try {
    const raw = localStorage.getItem(KEY);
    if (raw) settings.update((s) => ({ ...s, ...clean(JSON.parse(raw)) }));
  } catch {
    // ignore unreadable storage
  }
  settings.subscribe((s) => {
    try {
      const p: Persisted = {
        outFolderMode: s.outFolderMode,
        outFolder: s.outFolder,
        onExist: s.onExist,
        autoReveal: s.autoReveal,
      };
      localStorage.setItem(KEY, JSON.stringify(p));
    } catch {
      // ignore unwritable storage
    }
  });
}
