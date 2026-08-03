import { writable, type Writable } from "svelte/store";
import type { VideoInfo, ImageInfo } from "./wails";

// Source is the single picked input: either one video (trimmed) or an ordered
// set of images (sequenced). A null source is the empty state.
export type Source =
  | { kind: "video"; info: VideoInfo }
  | { kind: "images"; items: ImageInfo[] };

export const source: Writable<Source | null> = writable<Source | null>(null);
