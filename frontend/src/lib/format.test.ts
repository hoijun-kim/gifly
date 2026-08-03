import { describe, it, expect } from 'vitest';
import { msToTimecode, fpsFromFrameMs, frameMsFromFps, aspectHeight, humanBytes, estimateBytes } from './format';

describe('format', () => {
  it('formats ms as mm:ss.mmm', () => {
    expect(msToTimecode(0)).toBe('00:00.000');
    expect(msToTimecode(65432)).toBe('01:05.432');
  });
  it('converts between fps and frame ms', () => {
    expect(fpsFromFrameMs(100)).toBe(10);
    expect(frameMsFromFps(20)).toBe(50);
  });
  it('derives an even aspect height', () => {
    expect(aspectHeight(1600, 900, 800)).toBe(450);
    expect(aspectHeight(100, 101, 100)).toBe(102); // odd -> even
    expect(aspectHeight(1000, 1, 1)).toBe(2); // round(1*1/1000)=0 -> clamped to 2
    expect(aspectHeight(0, 0, 0)).toBe(2); // degenerate guard
  });
  it('formats bytes', () => {
    expect(humanBytes(1536)).toBe('1.5 KB');
  });
  it('estimates output bytes as a ballpark heuristic', () => {
    // Degenerate inputs (any non-positive dimension) -> 0.
    expect(estimateBytes(0, 480, 10, 256, true)).toBe(0);
    expect(estimateBytes(640, 0, 10, 256, true)).toBe(0);
    expect(estimateBytes(640, 480, 0, 256, true)).toBe(0);
    expect(estimateBytes(640, 480, -1, 256, true)).toBe(0);

    // Grows with frame count, all else equal.
    const fewFrames = estimateBytes(640, 480, 5, 128, false);
    const moreFrames = estimateBytes(640, 480, 20, 128, false);
    expect(moreFrames).toBeGreaterThan(fewFrames);

    // Grows with palette size, all else equal.
    const fewColors = estimateBytes(640, 480, 10, 16, false);
    const moreColors = estimateBytes(640, 480, 10, 256, false);
    expect(moreColors).toBeGreaterThan(fewColors);

    // Dithering adds noise that compresses worse -> larger estimate.
    const noDither = estimateBytes(640, 480, 10, 128, false);
    const withDither = estimateBytes(640, 480, 10, 128, true);
    expect(withDither).toBeGreaterThan(noDither);
  });
});
