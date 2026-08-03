<script lang="ts">
  // The Video mode screen: pick a video, trim it, set fps/size/loop/quality,
  // convert, watch progress, and land on the result card. All the actual
  // number crunching (timecodes, aspect math, validation, size estimate)
  // lives in ../lib/format and ../lib/validate; this component just wires
  // state to the bound Go methods in ../lib/wails. Shared visual language
  // (sections, fields, chips, buttons, progress, result card) lives in
  // ../style.css so this stays consistent with ../modes/Images.svelte.
  import { onDestroy } from "svelte";
  import { PickVideo, ConvertVideo, Cancel, RevealOutput, onProgress } from "../lib/wails";
  import type { VideoInfo, VideoRequest, ConvertResult } from "../lib/wails";
  import { msToTimecode, aspectHeight, humanBytes, estimateBytes } from "../lib/format";
  import { videoValid } from "../lib/validate";

  const FPS_PRESETS = [10, 15, 24, 30];
  const WIDTH_PRESETS = [240, 360, 480, 640];

  let video: VideoInfo | null = null;
  let picking = false;
  let pickError: string | null = null;

  let startSec = 0;
  let endSec = 0;
  let fps = 15;
  let width = 0;
  let loopChoice = "forever"; // "forever" | "once" | "custom"
  let loopCount = 2;
  let colors = 256;
  let dither = true;
  let targetMB = 0; // 0 or blank = no target

  let converting = false;
  let progressPhase = "";
  let progressPercent = 0;
  let stopProgress: (() => void) | null = null;

  let result: ConvertResult | null = null;
  let convertError: string | null = null;
  let previewUrl = "";

  $: durationSec = video ? video.DurationMS / 1000 : 0;
  $: startMs = Math.round(startSec * 1000);
  $: endMs = Math.round(endSec * 1000);
  $: outHeight = video && width > 0 ? aspectHeight(video.Width, video.Height, width) : 0;
  $: loopValue = loopChoice === "custom" ? String(Math.max(1, Math.round(loopCount))) : loopChoice;
  $: validationMessage = video ? videoValid({ startMs, endMs, fps, width }) : null;
  $: canConvert = !!video && !picking && !converting && !validationMessage;

  // Blank/NaN/negative target inputs all read as "off".
  $: targetMBSafe = Number.isFinite(targetMB) && targetMB > 0 ? targetMB : 0;
  $: targetBytes = targetMBSafe > 0 ? Math.round(targetMBSafe * 1024 * 1024) : 0;
  $: estFrames = fps > 0 && endMs > startMs ? Math.round((fps * (endMs - startMs)) / 1000) : 0;
  $: rawEstimate = estimateBytes(Math.round(width), outHeight, estFrames, Math.round(colors), dither);
  $: displayEstimate = targetBytes > 0 ? Math.min(rawEstimate, targetBytes) : rawEstimate;
  $: estimateCapped = targetBytes > 0 && rawEstimate > targetBytes;

  function basename(path: string): string {
    const idx = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
    return idx >= 0 ? path.slice(idx + 1) : path;
  }

  // Derives an output .gif path next to the source: swaps the extension
  // (or appends one) without touching the directory.
  function deriveOutPath(input: string): string {
    const lastSlash = Math.max(input.lastIndexOf("\\"), input.lastIndexOf("/"));
    const lastDot = input.lastIndexOf(".");
    const base = lastDot > lastSlash ? input.slice(0, lastDot) : input;
    return `${base}.gif`;
  }

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  function clampStart() {
    if (startSec < 0) startSec = 0;
    if (startSec > durationSec) startSec = durationSec;
  }

  function clampEnd() {
    if (endSec < 0) endSec = 0;
    if (endSec > durationSec) endSec = durationSec;
  }

  async function pickVideo() {
    pickError = null;
    picking = true;
    try {
      const info = await PickVideo();
      video = info;
      startSec = 0;
      endSec = info.DurationMS / 1000;
      fps = 15;
      width = info.Width;
      loopChoice = "forever";
      loopCount = 2;
      colors = 256;
      dither = true;
      targetMB = 0;
      result = null;
      convertError = null;
    } catch (err) {
      pickError = errorMessage(err);
    } finally {
      picking = false;
    }
  }

  async function convert() {
    if (!video || validationMessage) return;
    convertError = null;
    result = null;
    converting = true;
    progressPhase = "encoding";
    progressPercent = 0;
    stopProgress = onProgress((event) => {
      progressPhase = event.Phase;
      progressPercent = event.Percent;
    });

    const req: VideoRequest = {
      Input: video.Path,
      StartMS: startMs,
      EndMS: endMs,
      FPS: Math.round(fps),
      Width: Math.round(width),
      Loop: loopValue,
      Colors: Math.round(colors),
      Dither: dither,
      Out: deriveOutPath(video.Path),
      TargetKB: Math.round(targetMBSafe * 1024),
    };

    try {
      const res = await ConvertVideo(req);
      result = res;
      previewUrl = `/preview.gif?t=${Date.now()}`;
    } catch (err) {
      convertError = errorMessage(err);
    } finally {
      converting = false;
      if (stopProgress) {
        stopProgress();
        stopProgress = null;
      }
    }
  }

  function cancel() {
    Cancel();
  }

  function openFolder() {
    if (result) RevealOutput(result.Path);
  }

  // Back to the empty state so the user can pick a fresh source; the next
  // pickVideo() call resets all settings (fps/width/loop/quality/target).
  function makeAnother() {
    video = null;
    result = null;
    convertError = null;
    previewUrl = "";
  }

  // Switching to Images mode unmounts this component (App.svelte's
  // {#if $mode==='video'}), which would otherwise leave the convert:progress
  // listener registered and an in-flight backend job running unmanaged.
  onDestroy(() => {
    if (stopProgress) stopProgress();
    if (converting) Cancel();
  });
</script>

<div class="video-mode">
  {#if !video}
    <div class="pick-panel">
      <button class="btn-primary" type="button" on:click={pickVideo} disabled={picking}>
        {picking ? "Opening..." : "Pick a video"}
      </button>
      {#if pickError}<p class="error">{pickError}</p>{/if}
    </div>
  {:else}
    <section class="section">
      <div class="source-row">
        <span class="eyebrow">Source</span>
        <button class="btn-ghost" type="button" on:click={pickVideo} disabled={picking}>
          {picking ? "Opening..." : "Change"}
        </button>
      </div>
      <span class="source-name" title={video.Path}>{basename(video.Path)}</span>
      <dl class="meta">
        <div class="meta-item">
          <dt>Duration</dt>
          <dd class="mono">{msToTimecode(video.DurationMS)}</dd>
        </div>
        <div class="meta-item">
          <dt>Source size</dt>
          <dd class="mono">{video.Width}x{video.Height}</dd>
        </div>
      </dl>
      {#if pickError}<p class="error">{pickError}</p>{/if}
    </section>

    <section class="section">
      <span class="eyebrow">Timing</span>
      <div class="field-row">
        <label class="field">
          <span class="field-label">Start (s)</span>
          <input
            class="mono"
            type="number"
            min="0"
            max={durationSec}
            step="0.1"
            bind:value={startSec}
            on:blur={clampStart}
          />
          <span class="timecode mono">{msToTimecode(startMs)}</span>
        </label>
        <label class="field">
          <span class="field-label">End (s)</span>
          <input
            class="mono"
            type="number"
            min="0"
            max={durationSec}
            step="0.1"
            bind:value={endSec}
            on:blur={clampEnd}
          />
          <span class="timecode mono">{msToTimecode(endMs)}</span>
        </label>
      </div>
      <label class="field field-wide">
        <span class="field-label">FPS</span>
        <input class="mono" type="number" min="1" step="1" bind:value={fps} />
      </label>
      <div class="chip-row" role="group" aria-label="FPS presets">
        {#each FPS_PRESETS as preset (preset)}
          <button
            type="button"
            class="chip mono"
            class:active={Math.round(fps) === preset}
            on:click={() => (fps = preset)}
          >
            {preset}
          </button>
        {/each}
      </div>
    </section>

    <section class="section">
      <span class="eyebrow">Size</span>
      <div class="field-row">
        <label class="field">
          <span class="field-label">Width (px)</span>
          <input class="mono" type="number" min="2" step="1" bind:value={width} />
        </label>
        <div class="field">
          <span class="field-label">Height</span>
          <span class="mono readout">{outHeight}px</span>
        </div>
      </div>
      <div class="chip-row" role="group" aria-label="Width presets">
        {#each WIDTH_PRESETS as preset (preset)}
          <button
            type="button"
            class="chip mono"
            class:active={Math.round(width) === preset}
            on:click={() => (width = preset)}
          >
            {preset}
          </button>
        {/each}
      </div>
    </section>

    <section class="section">
      <span class="eyebrow">Loop</span>
      <div class="field-row">
        <label class="field">
          <span class="field-label">Repeat</span>
          <select bind:value={loopChoice}>
            <option value="forever">Forever</option>
            <option value="once">Once</option>
            <option value="custom">Custom</option>
          </select>
        </label>
        {#if loopChoice === "custom"}
          <label class="field">
            <span class="field-label">Times</span>
            <input class="mono" type="number" min="1" step="1" bind:value={loopCount} />
          </label>
        {/if}
      </div>
    </section>

    <section class="section">
      <span class="eyebrow">Quality</span>
      <label class="field field-wide">
        <span class="field-label">Colors <span class="mono">{colors}</span></span>
        <input type="range" min="2" max="256" step="1" bind:value={colors} />
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={dither} />
        <span>Dither</span>
      </label>
    </section>

    <section class="section">
      <span class="eyebrow">Output</span>
      <label class="field field-wide">
        <span class="field-label">Target size (MB) <span class="field-optional">optional</span></span>
        <input class="mono" type="number" min="0" step="0.1" placeholder="off" bind:value={targetMB} />
      </label>
      {#if targetMBSafe > 0}
        <p class="hint">gifly shrinks the width to fit ~{targetMBSafe} MB</p>
      {/if}
      <p class="estimate">
        <span>Estimated size</span>
        <span class="estimate-value mono">~{humanBytes(displayEstimate)}</span>
        {#if estimateCapped}<span class="estimate-note">(capped by target)</span>{/if}
      </p>
    </section>

    <div class="actions">
      <button class="btn-primary" type="button" on:click={convert} disabled={!canConvert}>
        {converting ? "Converting..." : "Make GIF"}
      </button>
      {#if validationMessage}
        <p class="hint">{validationMessage}</p>
      {/if}
    </div>

    {#if converting}
      <div class="progress">
        <div class="progress-track">
          <div class="progress-fill" style="width:{progressPercent}%"></div>
        </div>
        <div class="progress-row">
          <span class="mono">{progressPercent}%</span>
          <span class="progress-phase">{progressPhase}</span>
          <button class="btn-ghost" type="button" on:click={cancel}>Cancel</button>
        </div>
      </div>
    {/if}

    {#if convertError}
      <p class="error">{convertError}</p>
    {/if}

    {#if result}
      <section class="section result-card">
        <img class="preview" src={previewUrl} alt="Converted GIF preview" />
        <div class="result-meta">
          <span class="mono">{result.Width}x{result.Height}</span>
          <span class="mono">{humanBytes(result.Bytes)}</span>
        </div>
        <div class="btn-row">
          <button class="btn-ghost" type="button" on:click={openFolder}>Open folder</button>
          <button class="btn-ghost" type="button" on:click={makeAnother}>Make another</button>
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .video-mode {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
</style>
