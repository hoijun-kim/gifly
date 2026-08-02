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
  let h = Math.round((outW * srcH) / srcW);
  if (h < 2) h = 2;
  if (h % 2) h++; // if h is odd
  return h;
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
