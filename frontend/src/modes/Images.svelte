<script lang="ts">
  // The Images mode screen: add images, put them in order, set the
  // per-frame delay/size/loop/quality, convert, watch progress, and land
  // on the result card. Mirrors ../modes/Video.svelte's layout and flow;
  // the actual math (fps, aspect, validation) lives in ../lib/format and
  // ../lib/validate, this component just wires state to the bound Go
  // methods in ../lib/wails.
  import { onDestroy } from "svelte";
  import { PickImages, ConvertImages, Cancel, RevealOutput, onProgress } from "../lib/wails";
  import type { ImageInfo, ImagesRequest, ConvertResult } from "../lib/wails";
  import { fpsFromFrameMs, aspectHeight, humanBytes } from "../lib/format";
  import { imagesValid } from "../lib/validate";

  let images: ImageInfo[] = [];
  let picking = false;
  let pickError: string | null = null;
  let thumbErrors: Record<string, boolean> = {};

  let frameMs = 100;
  let width = 480;
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

  $: fps = fpsFromFrameMs(frameMs);
  $: outHeight = images.length > 0 && width > 0 ? aspectHeight(images[0].Width, images[0].Height, width) : 0;
  $: loopValue = loopChoice === "custom" ? String(Math.max(1, Math.round(loopCount))) : loopChoice;
  $: validationMessage = imagesValid({ count: images.length, frameMs, width });
  $: canConvert = images.length > 0 && !picking && !converting && !validationMessage;

  function basename(path: string): string {
    const idx = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
    return idx >= 0 ? path.slice(idx + 1) : path;
  }

  function extOf(path: string): string {
    const idx = path.lastIndexOf(".");
    return idx >= 0 && idx < path.length - 1 ? path.slice(idx + 1).toUpperCase() : "IMG";
  }

  // Best-effort local file preview - some WebView2 configurations block
  // file:// fetches from the http://wails.localhost origin, so a broken
  // thumbnail just falls back to the extension badge (see onThumbError).
  function fileUrl(path: string): string {
    const normalized = path.replace(/\\/g, "/");
    const prefixed = normalized.startsWith("/") ? normalized : `/${normalized}`;
    return `file://${encodeURI(prefixed)}`;
  }

  function onThumbError(path: string) {
    thumbErrors = { ...thumbErrors, [path]: true };
  }

  // Derives an output .gif path next to the first source image, named with
  // a timestamp so repeated conversions in the same folder do not collide.
  function deriveOutPath(firstInput: string): string {
    const lastSlash = Math.max(firstInput.lastIndexOf("\\"), firstInput.lastIndexOf("/"));
    const dir = lastSlash >= 0 ? firstInput.slice(0, lastSlash + 1) : "";
    return `${dir}gifly-${Date.now()}.gif`;
  }

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  function moveUp(i: number) {
    if (i <= 0 || i >= images.length) return;
    const next = images.slice();
    [next[i - 1], next[i]] = [next[i], next[i - 1]];
    images = next;
  }

  function moveDown(i: number) {
    if (i < 0 || i >= images.length - 1) return;
    const next = images.slice();
    [next[i], next[i + 1]] = [next[i + 1], next[i]];
    images = next;
  }

  function removeAt(i: number) {
    images = images.filter((_, idx) => idx !== i);
  }

  async function pickImages() {
    pickError = null;
    picking = true;
    try {
      const picked = await PickImages();
      images = [...images, ...picked];
    } catch (err) {
      pickError = errorMessage(err);
    } finally {
      picking = false;
    }
  }

  async function convert() {
    if (images.length === 0 || validationMessage) return;
    convertError = null;
    result = null;
    converting = true;
    progressPhase = "encoding";
    progressPercent = 0;
    stopProgress = onProgress((event) => {
      progressPhase = event.Phase;
      progressPercent = event.Percent;
    });

    const req: ImagesRequest = {
      Inputs: images.map((img) => img.Path),
      FrameMS: Math.round(frameMs),
      Width: Math.round(width),
      Loop: loopValue,
      Colors: Math.round(colors),
      Dither: dither,
      Out: deriveOutPath(images[0].Path),
    };

    try {
      const res = await ConvertImages(req);
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

  // Switching to Video mode unmounts this component (App.svelte's
  // {#if $mode==='video'}...{:else}...{/if}), which would otherwise leave
  // the convert:progress listener registered and an in-flight backend job
  // running unmanaged.
  onDestroy(() => {
    if (stopProgress) stopProgress();
    if (converting) Cancel();
  });
</script>

<div class="images-mode">
  {#if images.length === 0}
    <div class="pick-panel">
      <button class="btn-primary" type="button" on:click={pickImages} disabled={picking}>
        {picking ? "Opening..." : "Add images"}
      </button>
      {#if pickError}<p class="error">{pickError}</p>{/if}
    </div>
  {:else}
    <section class="panel">
      <div class="source-row">
        <span class="panel-title">Images <span class="mono">({images.length})</span></span>
        <button class="btn-ghost" type="button" on:click={pickImages} disabled={picking}>
          {picking ? "Opening..." : "Add images"}
        </button>
      </div>
      {#if pickError}<p class="error">{pickError}</p>{/if}
      <ol class="list">
        {#each images as img, i (img.Path + i)}
          <li class="row">
            <div class="thumb">
              {#if !thumbErrors[img.Path]}
                <img src={fileUrl(img.Path)} alt="" on:error={() => onThumbError(img.Path)} />
              {:else}
                <span class="thumb-fallback mono">{extOf(img.Path)}</span>
              {/if}
            </div>
            <div class="row-info">
              <span class="row-name" title={img.Path}>{basename(img.Path)}</span>
              <span class="row-dims mono">{img.Width}x{img.Height}</span>
            </div>
            <div class="row-actions">
              <button
                class="btn-icon"
                type="button"
                on:click={() => moveUp(i)}
                disabled={i === 0}
                aria-label="Move up"
              >
                Up
              </button>
              <button
                class="btn-icon"
                type="button"
                on:click={() => moveDown(i)}
                disabled={i === images.length - 1}
                aria-label="Move down"
              >
                Down
              </button>
              <button class="btn-icon" type="button" on:click={() => removeAt(i)} aria-label="Remove">
                Remove
              </button>
            </div>
          </li>
        {/each}
      </ol>
    </section>

    <section class="panel">
      <h3 class="panel-title">Frames</h3>
      <div class="field-row">
        <label class="field">
          <span class="field-label">Duration (ms)</span>
          <input class="mono" type="number" min="1" step="1" bind:value={frameMs} />
        </label>
        <div class="field">
          <span class="field-label">Rate</span>
          <span class="mono readout">= {fps.toFixed(1)} fps</span>
        </div>
      </div>
    </section>

    <section class="panel">
      <h3 class="panel-title">Output</h3>
      <div class="field-row">
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
  .images-mode {
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

  .list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: 260px;
    overflow-y: auto;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: var(--r);
  }

  .thumb {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    width: 40px;
    height: 40px;
    overflow: hidden;
    background: var(--bg);
    border: 1px solid var(--line);
    border-radius: 6px;
  }

  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .thumb-fallback {
    font-size: 10px;
    color: var(--faint);
  }

  .row-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1 1 auto;
    min-width: 0;
  }

  .row-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
    color: var(--text);
  }

  .row-dims {
    font-size: 11px;
    color: var(--faint);
  }

  .row-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 4px;
  }

  .btn-icon {
    padding: 5px 8px;
    border: 1px solid var(--line);
    border-radius: var(--r);
    background: transparent;
    color: var(--text);
    font-family: var(--ui);
    font-size: 11px;
    cursor: pointer;
  }

  .btn-icon:hover:not(:disabled) {
    background: var(--panel);
  }

  .btn-icon:disabled {
    color: var(--faint);
    cursor: not-allowed;
    opacity: 0.5;
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
    .btn-ghost,
    .btn-icon {
      transition: background-color 120ms ease;
    }

    .progress-fill {
      transition: width 150ms ease;
    }
  }
</style>
