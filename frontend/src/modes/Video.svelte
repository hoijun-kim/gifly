<script lang="ts">
  // The Video mode screen: pick a video, trim it, set fps/size/loop/quality,
  // convert, watch progress, and land on the result card. All the actual
  // number crunching (timecodes, aspect math, validation) lives in
  // ../lib/format and ../lib/validate; this component just wires state to
  // the bound Go methods in ../lib/wails.
  import { PickVideo, ConvertVideo, Cancel, RevealOutput, onProgress } from "../lib/wails";
  import type { VideoInfo, VideoRequest, ConvertResult } from "../lib/wails";
  import { msToTimecode, aspectHeight, humanBytes } from "../lib/format";
  import { videoValid } from "../lib/validate";

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
    <section class="panel">
      <div class="source-row">
        <span class="source-name" title={video.Path}>{basename(video.Path)}</span>
        <button class="btn-ghost" type="button" on:click={pickVideo} disabled={picking}>
          {picking ? "Opening..." : "Change"}
        </button>
      </div>
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

    <section class="panel">
      <h3 class="panel-title">Trim</h3>
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
    </section>

    <section class="panel">
      <h3 class="panel-title">Output</h3>
      <div class="field-row">
        <label class="field">
          <span class="field-label">FPS</span>
          <input class="mono" type="number" min="1" step="1" bind:value={fps} />
        </label>
        <label class="field">
          <span class="field-label">Width</span>
          <input class="mono" type="number" min="2" step="1" bind:value={width} />
        </label>
        <div class="field">
          <span class="field-label">Height</span>
          <span class="mono readout">{outHeight}px</span>
        </div>
      </div>
      <div class="field-row">
        <label class="field">
          <span class="field-label">Loop</span>
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

    <section class="panel">
      <h3 class="panel-title">Quality</h3>
      <label class="field field-wide">
        <span class="field-label">Colors <span class="mono">{colors}</span></span>
        <input type="range" min="2" max="256" step="1" bind:value={colors} />
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={dither} />
        <span>Dither</span>
      </label>
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
      <section class="panel result">
        <img class="preview" src={previewUrl} alt="Converted GIF preview" />
        <div class="result-row">
          <div class="result-meta">
            <span class="mono">{result.Width}x{result.Height}</span>
            <span class="mono">{humanBytes(result.Bytes)}</span>
          </div>
          <button class="btn-ghost" type="button" on:click={openFolder}>Open folder</button>
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .video-mode {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .pick-panel {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 48px 24px;
    border: 1px dashed var(--line);
    border-radius: var(--r-panel);
    background: var(--panel);
  }

  .panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px;
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: var(--r-panel);
  }

  .panel-title {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--muted);
  }

  .source-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .source-name {
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    display: flex;
    gap: 24px;
    margin: 0;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .meta-item dt {
    font-size: 11px;
    color: var(--faint);
  }

  .meta-item dd {
    margin: 0;
    color: var(--text);
  }

  .field-row {
    display: flex;
    flex-wrap: wrap;
    gap: 14px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1 1 100px;
  }

  .field-wide {
    flex-basis: 100%;
  }

  .field-label {
    font-size: 12px;
    color: var(--muted);
  }

  .field input[type="number"],
  .field select {
    padding: 6px 8px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: var(--r);
    color: var(--text);
    font-family: var(--ui);
    font-size: 13px;
  }

  .field input[type="range"] {
    accent-color: var(--accent);
  }

  .readout {
    padding: 6px 0;
    color: var(--muted);
  }

  .timecode {
    color: var(--faint);
    font-size: 12px;
  }

  .toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    width: fit-content;
    color: var(--text);
    font-size: 13px;
  }

  .toggle input {
    accent-color: var(--accent);
  }

  .actions {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .hint {
    margin: 0;
    color: var(--muted);
    font-size: 12px;
  }

  .error {
    margin: 0;
    color: var(--stop);
    font-size: 13px;
  }

  .btn-primary {
    padding: 10px 16px;
    border: none;
    border-radius: var(--r);
    background: var(--accent);
    color: var(--accent-ink);
    font-family: var(--ui);
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-primary:disabled {
    background: var(--panel-2);
    color: var(--faint);
    cursor: not-allowed;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hi);
  }

  .btn-ghost {
    padding: 7px 12px;
    border: 1px solid var(--line);
    border-radius: var(--r);
    background: transparent;
    color: var(--text);
    font-family: var(--ui);
    font-size: 13px;
    cursor: pointer;
  }

  .btn-ghost:hover {
    background: var(--panel-2);
  }

  .progress {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .progress-track {
    height: 8px;
    border-radius: 999px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--accent);
  }

  .progress-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .progress-phase {
    flex: 1 1 auto;
    color: var(--muted);
    font-size: 12px;
    text-transform: capitalize;
  }

  .result {
    align-items: stretch;
  }

  .preview {
    width: 100%;
    max-height: 320px;
    object-fit: contain;
    border-radius: var(--r);
    background: var(--panel-2);
  }

  .result-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .result-meta {
    display: flex;
    gap: 16px;
  }

  @media (prefers-reduced-motion: no-preference) {
    .btn-primary,
    .btn-ghost {
      transition: background-color 120ms ease;
    }

    .progress-fill {
      transition: width 150ms ease;
    }
  }
</style>
