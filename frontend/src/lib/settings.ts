import { writable, type Writable } from "svelte/store";

export type Format = "gif" | "webp" | "apng";
export type Aspect = "" | "1:1" | "16:9" | "9:16";
export type DitherMethod = "none" | "bayer" | "sierra2" | "floyd";
export type LoopChoice = "forever" | "once" | "custom";

// Settings holds every SHARED output option (everything that does not depend on
// whether the source is a video or images). Source-specific timing (trim, fps,
// frame duration) is owned by the timing components, not here.
export interface Settings {
  format: Format;
  aspect: Aspect;
  width: number;
  fps: number; // used when the source is a video
  frameMs: number; // used when the source is images
  loopChoice: LoopChoice;
  loopCount: number;
  speed: number; // 0.25..4, 1 = normal
  reverse: boolean;
  boomerang: boolean;
  colors: number; // GIF palette size 2..256
  dither: DitherMethod; // GIF only
  webpQuality: number; // WebP only, 0..100
  targetMB: number; // 0 = no target
  preset: string; // "" | "discord" | "slack" | "twitter"
}

export function defaultSettings(): Settings {
  return {
    format: "gif",
    aspect: "",
    width: 480,
    fps: 15,
    frameMs: 100,
    loopChoice: "forever",
    loopCount: 2,
    speed: 1,
    reverse: false,
    boomerang: false,
    colors: 256,
    dither: "sierra2",
    webpQuality: 75,
    targetMB: 0,
    preset: "",
  };
}

export const settings: Writable<Settings> = writable<Settings>(defaultSettings());
