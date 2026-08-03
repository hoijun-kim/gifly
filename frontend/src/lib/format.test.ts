import { describe, it, expect } from 'vitest';
import { msToTimecode, fpsFromFrameMs, frameMsFromFps, aspectHeight, humanBytes, outputHeight, formatExt, formatLabel, estimateBytes } from './format';

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
});

describe('outputHeight', () => {
  it('free aspect follows the source ratio (even)', () => {
    expect(outputHeight(1600, 900, '', 800)).toBe(450);
  });
  it('1:1 is square, 16:9 and 9:16 follow the ratio (even)', () => {
    expect(outputHeight(1600, 900, '1:1', 200)).toBe(200);
    expect(outputHeight(1600, 900, '16:9', 200) % 2).toBe(0);
    expect(outputHeight(1600, 900, '9:16', 200)).toBe(356);
  });
  it('degenerate inputs floor at 2', () => {
    expect(outputHeight(0, 0, '', 0)).toBe(2);
  });
});

describe('format ext/label', () => {
  it('maps each format', () => {
    expect(formatExt('gif')).toBe('.gif');
    expect(formatExt('webp')).toBe('.webp');
    expect(formatExt('apng')).toBe('.png');
    expect(formatLabel('apng')).toBe('APNG');
  });
});

describe('estimateBytes', () => {
  const base = { width: 200, height: 200, frames: 20, colors: 256, dither: true, webpQuality: 75 };
  it('is zero for non-positive dimensions', () => {
    expect(estimateBytes({ ...base, format: 'gif', width: 0 })).toBe(0);
    expect(estimateBytes({ ...base, format: 'gif', height: 0 })).toBe(0);
    expect(estimateBytes({ ...base, format: 'gif', frames: 0 })).toBe(0);
  });
  it('orders apng > gif > webp for the same content', () => {
    const gif = estimateBytes({ ...base, format: 'gif' });
    const webp = estimateBytes({ ...base, format: 'webp' });
    const apng = estimateBytes({ ...base, format: 'apng' });
    expect(webp).toBeLessThan(gif);
    expect(apng).toBeGreaterThan(gif);
  });
  it('webp estimate grows with quality', () => {
    const lo = estimateBytes({ ...base, format: 'webp', webpQuality: 20 });
    const hi = estimateBytes({ ...base, format: 'webp', webpQuality: 95 });
    expect(hi).toBeGreaterThan(lo);
  });
  it('gif estimate grows with colors and dithering', () => {
    const fewColors = estimateBytes({ ...base, format: 'gif', colors: 16, dither: false });
    const moreColors = estimateBytes({ ...base, format: 'gif', colors: 256, dither: false });
    expect(moreColors).toBeGreaterThan(fewColors);
    const noDither = estimateBytes({ ...base, format: 'gif', colors: 128, dither: false });
    const withDither = estimateBytes({ ...base, format: 'gif', colors: 128, dither: true });
    expect(withDither).toBeGreaterThan(noDither);
  });
});
