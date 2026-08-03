import type { VideoInfo, ImageInfo, VideoRequest, ImagesRequest } from "./wails";
import type { Settings } from "./settings";
import type { Source } from "./source";

// PLATFORM_PRESETS set a max width and a size target for common chat platforms.
export const PLATFORM_PRESETS: Record<string, { label: string; maxWidth: number; targetMB: number }> = {
  discord: { label: "Discord", maxWidth: 480, targetMB: 8 },
  slack: { label: "Slack", maxWidth: 480, targetMB: 2 },
  twitter: { label: "Twitter", maxWidth: 506, targetMB: 15 },
};

// dirOf returns the directory part of a path (no trailing separator), or "" if
// the path has no directory. Windows and POSIX separators are both handled.
export function dirOf(path: string): string {
  const slash = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
  return slash >= 0 ? path.slice(0, slash) : "";
}

// baseNameNoExt returns a path's file name without its directory or extension.
export function baseNameNoExt(path: string): string {
  const slash = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
  const name = slash >= 0 ? path.slice(slash + 1) : path;
  const dot = name.lastIndexOf(".");
  return dot > 0 ? name.slice(0, dot) : name;
}

// defaultOutName is the base output name suggested for a source: a video's own
// name, or "gifly" for a set of images (or the empty state).
export function defaultOutName(src: Source | null): string {
  if (src && src.kind === "video") return baseNameNoExt(src.info.Path);
  return "gifly";
}

// loopValue turns the loop choice into the string the backend parses.
export function loopValue(s: Settings): string {
  return s.loopChoice === "custom" ? String(Math.max(1, Math.round(s.loopCount))) : s.loopChoice;
}

function targetKB(targetMB: number): number {
  const mb = Number.isFinite(targetMB) && targetMB > 0 ? targetMB : 0;
  return Math.round(mb * 1024);
}

// outDir picks the destination folder: a chosen custom folder, else the folder
// beside the source.
function outDir(s: Settings, sourcePath: string): string {
  if (s.outFolderMode === "custom" && s.outFolder.trim()) return s.outFolder.trim();
  return dirOf(sourcePath);
}

export function buildVideoRequest(info: VideoInfo, s: Settings, timing: { startMs: number; endMs: number }): VideoRequest {
  return {
    Input: info.Path,
    StartMS: timing.startMs,
    EndMS: timing.endMs,
    FPS: Math.round(s.fps),
    Width: Math.round(s.width),
    SrcWidth: info.Width,
    SrcHeight: info.Height,
    Aspect: s.aspect,
    Speed: s.speed,
    Reverse: s.reverse,
    Boomerang: s.boomerang,
    Loop: loopValue(s),
    Colors: Math.round(s.colors),
    Dither: s.dither,
    WebPQuality: Math.round(s.webpQuality),
    Format: s.format,
    OutDir: outDir(s, info.Path),
    OutName: s.outName.trim() || baseNameNoExt(info.Path),
    OnExist: s.onExist,
    TargetKB: targetKB(s.targetMB),
  };
}

export function buildImagesRequest(items: ImageInfo[], s: Settings): ImagesRequest {
  return {
    Inputs: items.map((i) => i.Path),
    FrameMS: Math.round(s.frameMs),
    Width: Math.round(s.width),
    Aspect: s.aspect,
    Speed: s.speed,
    Reverse: s.reverse,
    Boomerang: s.boomerang,
    Loop: loopValue(s),
    Colors: Math.round(s.colors),
    Dither: s.dither,
    WebPQuality: Math.round(s.webpQuality),
    Format: s.format,
    OutDir: outDir(s, items[0].Path),
    OutName: s.outName.trim() || "gifly",
    OnExist: s.onExist,
    TargetKB: targetKB(s.targetMB),
  };
}
