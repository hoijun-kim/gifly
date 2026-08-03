import type { Format, Aspect } from "./settings";

export function msToTimecode(ms: number): string {
  const mm = Math.floor(ms / 60000);
  const remaining = ms % 60000;
  const ss = Math.floor(remaining / 1000);
  const mmm = remaining % 1000;
  return `${String(mm).padStart(2, '0')}:${String(ss).padStart(2, '0')}.${String(mmm).padStart(3, '0')}`;
}

export function fpsFromFrameMs(ms: number): number {
  return 1000 / ms;
}

export function frameMsFromFps(fps: number): number {
  return 1000 / fps;
}

export function aspectHeight(srcW: number, srcH: number, outW: number): number {
  if (srcW <= 0 || srcH <= 0 || outW <= 0) return 2;
  let h = Math.round((outW * srcH) / srcW);
  if (h < 2) h = 2;
  if (h % 2) h++; // if h is odd
  return h;
}

// outputHeight mirrors Go's gifjob.OutputHeight: for the presets the height
// follows the target ratio; for free it follows the source ratio. Even, min 2.
export function outputHeight(srcW: number, srcH: number, aspect: Aspect, outW: number): number {
  if (outW <= 0) return 2;
  let h: number;
  switch (aspect) {
    case "1:1":
      h = outW;
      break;
    case "16:9":
      h = Math.round((outW * 9) / 16);
      break;
    case "9:16":
      h = Math.round((outW * 16) / 9);
      break;
    default:
      if (srcW <= 0 || srcH <= 0) return 2;
      h = Math.round((outW * srcH) / srcW);
  }
  if (h < 2) h = 2;
  if (h % 2) h++;
  return h;
}

export function formatExt(format: Format): string {
  if (format === "webp") return ".webp";
  if (format === "apng") return ".png";
  return ".gif";
}

export function formatLabel(format: Format): string {
  if (format === "webp") return "WebP";
  if (format === "apng") return "APNG";
  return "GIF";
}

// estimateBytes is a rough live readout, not a byte-accurate prediction (real
// size depends on frame content this function cannot see). GIF uses a palette-
// and dither-scaled bits-per-pixel; WebP is lossy (bpp scales with quality) and
// smaller; APNG is lossless and larger.
export function estimateBytes(o: {
  format: Format;
  width: number;
  height: number;
  frames: number;
  colors: number;
  dither: boolean;
  webpQuality: number;
}): number {
  if (o.width <= 0 || o.height <= 0 || o.frames <= 0) return 0;
  let bpp: number;
  switch (o.format) {
    case "webp":
      bpp = 0.06 + (Math.max(0, Math.min(100, o.webpQuality)) / 100) * 0.34;
      break;
    case "apng":
      bpp = 0.9;
      break;
    default:
      bpp = (0.12 + (o.colors / 256) * 0.45) * (o.dither ? 1.15 : 1.0);
  }
  return Math.round(o.frames * o.width * o.height * bpp);
}

export function humanBytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = n;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  if (unitIndex === 0) {
    return `${Math.round(size)} ${units[unitIndex]}`;
  } else {
    return `${size.toFixed(1)} ${units[unitIndex]}`;
  }
}
