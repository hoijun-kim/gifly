import { describe, it, expect } from "vitest";
import { defaultSettings } from "./settings";

describe("defaultSettings", () => {
  it("starts as a dithered 256-color GIF at normal speed, no crop, forever loop", () => {
    const s = defaultSettings();
    expect(s.format).toBe("gif");
    expect(s.aspect).toBe("");
    expect(s.colors).toBe(256);
    expect(s.dither).toBe("sierra2");
    expect(s.webpQuality).toBe(75);
    expect(s.speed).toBe(1);
    expect(s.reverse).toBe(false);
    expect(s.boomerang).toBe(false);
    expect(s.loopChoice).toBe("forever");
    expect(s.width).toBe(480);
    expect(s.fps).toBe(15);
    expect(s.frameMs).toBe(100);
    expect(s.targetMB).toBe(0);
    expect(s.preset).toBe("");
  });
});
