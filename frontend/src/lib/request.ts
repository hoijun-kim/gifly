import type { VideoInfo, ImageInfo, VideoRequest, ImagesRequest } from "./wails";
import type { Settings, Format } from "./settings";
import { formatExt } from "./format";

// PLATFORM_PRESETS set a max width and a size target for common chat platforms.
export const PLATFORM_PRESETS: Record<string, { label: string; maxWidth: number; targetMB: number }> = {
  discord: { label: "Discord", maxWidth: 480, targetMB: 8 },
  slack: { label: "Slack", maxWidth: 480, targetMB: 2 },
  twitter: { label: "Twitter", maxWidth: 506, targetMB: 15 },
};

// swapExt replaces (or appends) the extension of a path without touching its
// directory. Windows and POSIX separators are both handled.
function swapExt(path: string, ext: string): string {
  const slash = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
  const dot = path.lastIndexOf(".");
  const base = dot > slash ? path.slice(0, dot) : path;
  return base + ext;
}

// deriveOut names a video's output next to the source with the format's ext.
export function deriveOut(inputPath: string, format: Format): string {
  return swapExt(inputPath, formatExt(format));
}

// deriveImagesOut names an images output next to the first image with a
// timestamp (so repeated runs in one folder do not collide) and the format ext.
export function deriveImagesOut(firstPath: string, format: Format): string {
  const slash = Math.max(firstPath.lastIndexOf("\\"), firstPath.lastIndexOf("/"));
  const dir = slash >= 0 ? firstPath.slice(0, slash + 1) : "";
  return `${dir}gifly-${Date.now()}${formatExt(format)}`;
}

// loopValue turns the loop choice into the string the backend parses.
export function loopValue(s: Settings): string {
  return s.loopChoice === "custom" ? String(Math.max(1, Math.round(s.loopCount))) : s.loopChoice;
}

function targetKB(targetMB: number): number {
  const mb = Number.isFinite(targetMB) && targetMB > 0 ? targetMB : 0;
  return Math.round(mb * 1024);
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
    Out: deriveOut(info.Path, s.format),
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
    Out: deriveImagesOut(items[0].Path, s.format),
    TargetKB: targetKB(s.targetMB),
  };
}
