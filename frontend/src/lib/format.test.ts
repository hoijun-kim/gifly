import { describe, it, expect } from 'vitest';
import { msToTimecode, fpsFromFrameMs, frameMsFromFps, aspectHeight, humanBytes } from './format';

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
  });
  it('formats bytes', () => {
    expect(humanBytes(1536)).toBe('1.5 KB');
  });
});
