import { describe, it, expect } from "vitest";
import { buildVideoRequest, buildImagesRequest, deriveOut, PLATFORM_PRESETS } from "./request";
import { defaultSettings } from "./settings";

describe("buildVideoRequest", () => {
  it("carries every option and derives the output extension from the format", () => {
    const info = { Path: "C:/clips/a.mp4", DurationMS: 3000, Width: 1280, Height: 720, FPS: 30 };
    const s = { ...defaultSettings(), format: "webp" as const, aspect: "1:1" as const, width: 320, fps: 24, speed: 2, reverse: true, boomerang: true, colors: 128, dither: "bayer" as const, webpQuality: 82, loopChoice: "once" as const, targetMB: 5 };
    const req = buildVideoRequest(info, s, { startMs: 100, endMs: 2500 });
    expect(req.Input).toBe("C:/clips/a.mp4");
    expect(req.Out.endsWith(".webp")).toBe(true);
    expect(req.Format).toBe("webp");
    expect(req.Aspect).toBe("1:1");
    expect(req.SrcWidth).toBe(1280);
    expect(req.SrcHeight).toBe(720);
    expect(req.Speed).toBe(2);
    expect(req.Reverse).toBe(true);
    expect(req.Boomerang).toBe(true);
    expect(req.FPS).toBe(24);
    expect(req.Width).toBe(320);
    expect(req.Dither).toBe("bayer");
    expect(req.WebPQuality).toBe(82);
    expect(req.Loop).toBe("once");
    expect(req.StartMS).toBe(100);
    expect(req.EndMS).toBe(2500);
    expect(req.TargetKB).toBe(Math.round(5 * 1024));
  });
});

describe("buildImagesRequest", () => {
  it("maps the ordered paths, frame duration and a timestamped output", () => {
    const items = [
      { Path: "C:/pics/1.png", Width: 100, Height: 80 },
      { Path: "C:/pics/2.png", Width: 100, Height: 80 },
    ];
    const s = { ...defaultSettings(), format: "apng" as const, frameMs: 120, width: 400 };
    const req = buildImagesRequest(items, s);
    expect(req.Inputs).toEqual(["C:/pics/1.png", "C:/pics/2.png"]);
    expect(req.FrameMS).toBe(120);
    expect(req.Format).toBe("apng");
    expect(req.Out.endsWith(".png")).toBe(true);
  });
});

describe("presets", () => {
  it("defines discord, slack and twitter with a max width and target MB", () => {
    for (const key of ["discord", "slack", "twitter"]) {
      expect(PLATFORM_PRESETS[key].maxWidth).toBeGreaterThan(0);
      expect(PLATFORM_PRESETS[key].targetMB).toBeGreaterThan(0);
    }
  });
});

describe("deriveOut", () => {
  it("swaps the extension to match the format", () => {
    expect(deriveOut("C:/x/a.mp4", "gif")).toBe("C:/x/a.gif");
    expect(deriveOut("C:/x/a.mov", "webp")).toBe("C:/x/a.webp");
  });
});
