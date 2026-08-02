export function videoValid(v: { startMs: number; endMs: number; fps: number; width: number }): string | null {
  if (v.width <= 0) {
    return 'Output width must be positive';
  }
  if (v.endMs <= v.startMs) {
    return 'End time must be greater than start time';
  }
  if (v.fps <= 0) {
    return 'FPS must be greater than 0';
  }
  return null;
}

export function imagesValid(v: { count: number; frameMs: number; width: number }): string | null {
  if (v.width <= 0) {
    return 'Output width must be positive';
  }
  if (v.count <= 0) {
    return 'Count must be greater than 0';
  }
  if (v.frameMs <= 0) {
    return 'Frame duration must be greater than 0';
  }
  return null;
}
