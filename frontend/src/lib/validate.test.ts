import { describe, it, expect } from 'vitest';
import { videoValid, imagesValid } from './validate';

describe('validate', () => {
  it('accepts a good video config and rejects bad ones', () => {
    expect(videoValid({ startMs: 0, endMs: 2000, fps: 15, width: 480 })).toBeNull();
    expect(videoValid({ startMs: 2000, endMs: 2000, fps: 15, width: 480 })).not.toBeNull(); // end<=start
    expect(videoValid({ startMs: 0, endMs: 2000, fps: 0, width: 480 })).not.toBeNull();
    expect(videoValid({ startMs: 0, endMs: 2000, fps: 15, width: 0 })).not.toBeNull();
  });
  it('accepts images and rejects empty/zero', () => {
    expect(imagesValid({ count: 2, frameMs: 100, width: 400 })).toBeNull();
    expect(imagesValid({ count: 0, frameMs: 100, width: 400 })).not.toBeNull();
    expect(imagesValid({ count: 2, frameMs: 0, width: 400 })).not.toBeNull();
    expect(imagesValid({ count: 2, frameMs: 100, width: 0 })).not.toBeNull();
  });
});
