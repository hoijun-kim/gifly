import { describe, it, expect } from "vitest";
import { buildVideoRequest, buildImagesRequest, dirOf, baseNameNoExt, defaultOutName, PLATFORM_PRESETS } from "./request";
import { defaultSettings } from "./settings";

describe("path helpers", () => {
  it("splits directory and base name", () => {
    expect(dirOf("C:/clips/a.mp4")).toBe("C:/clips");
    expect(dirOf("C:\\clips\\a.mp4")).toBe("C:\\clips");
    expect(dirOf("a.mp4")).toBe("");
    expect(baseNameNoExt("C:/clips/a.mp4")).toBe("a");
    expect(baseNameNoExt("C:/clips/my.clip.mov")).toBe("my.clip");
  });
  it("derives a default output name", () => {
    expect(defaultOutName({ kind: "video", info: { Path: "C:/x/sunset.mp4", DurationMS: 1, Width: 1, Height: 1, FPS: 1 } })).toBe("sunset");
    expect(defaultOutName({ kind: "images", items: [] })).toBe("gifly");
    expect(defaultOutName(null)).toBe("gifly");
  });
});

describe("buildVideoRequest", () => {
  it("carries every option, derives dir/name beside the source, keeps the on-exist policy", () => {
    const info = { Path: "C:/clips/a.mp4", DurationMS: 3000, Width: 1280, Height: 720, FPS: 30 };
    const s = { ...defaultSettings(), format: "webp" as const, aspect: "1:1" as const, width: 320, fps: 24, speed: 2, reverse: true, boomerang: true, colors: 128, dither: "bayer" as const, webpQuality: 82, loopChoice: "once" as const, targetMB: 5, onExist: "timestamp" as const };
    const req = buildVideoRequest(info, s, { startMs: 100, endMs: 2500 });
    expect(req.Input).toBe("C:/clips/a.mp4");
    expect(req.OutDir).toBe("C:/clips");
    expect(req.OutName).toBe("a");
    expect(req.OnExist).toBe("timestamp");
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
  it("uses a custom folder and name override when set", () => {
    const info = { Path: "C:/clips/a.mp4", DurationMS: 3000, Width: 100, Height: 100, FPS: 30 };
    const s = { ...defaultSettings(), outFolderMode: "custom" as const, outFolder: "D:/exports", outName: "  my clip  " };
    const req = buildVideoRequest(info, s, { startMs: 0, endMs: 1000 });
    expect(req.OutDir).toBe("D:/exports");
    expect(req.OutName).toBe("my clip");
  });
});

describe("buildImagesRequest", () => {
  it("maps the ordered paths, defaults name to gifly, dir beside the first image", () => {
    const items = [
      { Path: "C:/pics/1.png", Width: 100, Height: 80 },
      { Path: "C:/pics/2.png", Width: 100, Height: 80 },
    ];
    const s = { ...defaultSettings(), format: "apng" as const, frameMs: 120, width: 400 };
    const req = buildImagesRequest(items, s);
    expect(req.Inputs).toEqual(["C:/pics/1.png", "C:/pics/2.png"]);
    expect(req.FrameMS).toBe(120);
    expect(req.Format).toBe("apng");
    expect(req.OutDir).toBe("C:/pics");
    expect(req.OutName).toBe("gifly");
    expect(req.OnExist).toBe("number");
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
