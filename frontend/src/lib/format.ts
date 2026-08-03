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

// Ballpark output-size estimate for the live "~N MB estimated" readout.
// This is a heuristic, not a byte-accurate prediction: real GIF size depends
// on frame content (motion, gradients, noise) that this function does not
// see. bpp scales with the palette size (more colors -> less LZW-friendly
// runs -> more bytes/pixel) and dithering (adds noise, which compresses
// worse).
export function estimateBytes(width: number, height: number, frames: number, colors: number, dither: boolean): number {
  if (width <= 0 || height <= 0 || frames <= 0) return 0;
  const bpp = (0.12 + (colors / 256) * 0.45) * (dither ? 1.15 : 1.0);
  return Math.round(frames * width * height * bpp);
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
