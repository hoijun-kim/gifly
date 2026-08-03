<script lang="ts">
  // The single unified screen, laid out as two panes: a fixed left STAGE
  // (preview + source summary + the primary action) and a scrolling right
  // PANEL of option cards. All pure logic (request building, validation,
  // estimate) lives in ../lib; this orchestrator wires the source/settings
  // stores to the bound Go methods in ../lib/wails.
  import { onDestroy } from "svelte";
  import { source } from "../lib/source";
  import { settings } from "../lib/settings";
  import { ConvertVideo, ConvertImages, Cancel, RevealOutput, onProgress } from "../lib/wails";
  import type { ConvertResult } from "../lib/wails";
  import { humanBytes, formatLabel, outputHeight, estimateBytes } from "../lib/format";
  import { videoValid, imagesValid, settingsValid } from "../lib/validate";
  import { buildVideoRequest, buildImagesRequest } from "../lib/request";
  import SourcePanel from "./SourcePanel.svelte";
  import VideoTiming from "./VideoTiming.svelte";
  import ImagesTiming from "./ImagesTiming.svelte";
  import FormatPicker from "./FormatPicker.svelte";
  import SizeControls from "./SizeControls.svelte";
  import PlaybackControls from "./PlaybackControls.svelte";
  import QualityControls from "./QualityControls.svelte";
  import OutputControls from "./OutputControls.svelte";

  let startSec = 0;
  let endSec = 0;
  let trimInitPath = "";

  let converting = false;
  let progressPhase = "";
  let progressPercent = 0;
  let stopProgress: (() => void) | null = null;

  let result: ConvertResult | null = null;
  let convertError: string | null = null;
  let previewUrl = "";

  $: src = $source;

  // Initialize the trim window once per newly picked video (and clear any prior
  // result); switching away from a video resets the guard so the next video
  // re-initializes.
  $: if (src && src.kind === "video" && src.info.Path !== trimInitPath) {
    startSec = 0;
    endSec = src.info.DurationMS / 1000;
    trimInitPath = src.info.Path;
    result = null;
    convertError = null;
  }
  $: if ((!src || src.kind !== "video") && trimInitPath !== "") {
    trimInitPath = "";
  }

  $: durationSec = src && src.kind === "video" ? src.info.DurationMS / 1000 : 0;
  $: startMs = Math.round(startSec * 1000);
  $: endMs = Math.round(endSec * 1000);

  $: timingMessage =
    src && src.kind === "video"
      ? videoValid({ startMs, endMs, fps: $settings.fps, width: $settings.width })
      : src && src.kind === "images"
        ? imagesValid({ count: src.items.length, frameMs: $settings.frameMs, width: $settings.width })
        : null;
  $: validationMessage = settingsValid($settings) ?? timingMessage;
  $: canConvert = !!src && !converting && !validationMessage;

  // Live output-height and size-estimate from the source dims + settings.
  $: srcW = src && src.kind === "video" ? src.info.Width : src && src.kind === "images" ? (src.items[0]?.Width ?? 0) : 0;
  $: srcH = src && src.kind === "video" ? src.info.Height : src && src.kind === "images" ? (src.items[0]?.Height ?? 0) : 0;
  $: outHeight = outputHeight(srcW, srcH, $settings.aspect, $settings.width);
  $: frames =
    src && src.kind === "video"
      ? Math.round(($settings.fps * Math.max(0, endMs - startMs)) / 1000)
      : src && src.kind === "images"
        ? src.items.length
        : 0;
  $: rawEstimate = estimateBytes({
    format: $settings.format,
    width: Math.round($settings.width),
    height: outHeight,
    frames,
    colors: Math.round($settings.colors),
    dither: $settings.dither !== "none",
    webpQuality: $settings.webpQuality,
  });
  $: targetBytes = $settings.targetMB > 0 ? Math.round($settings.targetMB * 1024 * 1024) : 0;
  $: displayEstimate = targetBytes > 0 ? Math.min(rawEstimate, targetBytes) : rawEstimate;
  $: estimateCapped = targetBytes > 0 && rawEstimate > targetBytes;

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  async function convert() {
    const current = $source;
    if (!current || validationMessage) return;
    convertError = null;
    result = null;
    converting = true;
    progressPhase = "encoding";
    progressPercent = 0;
    stopProgress = onProgress((event) => {
      progressPhase = event.Phase;
      progressPercent = event.Percent;
    });
    try {
      let res: ConvertResult;
      if (current.kind === "video") {
        res = await ConvertVideo(buildVideoRequest(current.info, $settings, { startMs, endMs }));
      } else {
        res = await ConvertImages(buildImagesRequest(current.items, $settings));
      }
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

  function makeAnother() {
    source.set(null);
    result = null;
    convertError = null;
    previewUrl = "";
  }

  onDestroy(() => {
    if (stopProgress) stopProgress();
    if (converting) Cancel();
  });
</script>

{#if !src}
  <div class="body empty-body">
    <SourcePanel />
  </div>
{:else}
  <div class="body">
    <!-- LEFT STAGE -->
    <div class="stage">
      <div class="preview-frame">
        {#if result}
          <span class="badge ok">done</span>
          <img class="preview-img" src={previewUrl} alt="Converted preview" />
        {:else if converting}
          <span class="badge">encoding&hellip;</span>
          <div class="preview-placeholder">Encoding your {formatLabel($settings.format)}&hellip;</div>
        {:else}
          <span class="badge">preview</span>
          <div class="preview-placeholder">Your {formatLabel($settings.format)} preview appears here after you convert.</div>
        {/if}
      </div>

      <SourcePanel />

      {#if convertError}
        <p class="error">{convertError}</p>
      {/if}

      <div class="cta-wrap">
        {#if result}
          <div class="result-meta">
            <span class="mono">{result.Width}x{result.Height}</span>
            <span class="mono">{humanBytes(result.Bytes)}</span>
          </div>
          <div class="dz-actions">
            <button class="btn ghost block sm" type="button" on:click={openFolder}>Open folder</button>
            <button class="btn primary block sm" type="button" on:click={makeAnother}>Make another</button>
          </div>
        {:else if converting}
          <div class="prog">
            <div class="prog-track"><div class="prog-fill" style="width:{progressPercent}%"></div></div>
            <div class="prog-row">
              <span class="mono">{progressPercent}%</span>
              <span class="prog-phase">{progressPhase}</span>
              <button class="link" type="button" on:click={cancel}>Cancel</button>
            </div>
          </div>
        {:else}
          <div class="estimate-line">
            <span>Estimated size</span>
            <span class="estimate-val">
              <b>~{humanBytes(displayEstimate)}</b>
              {#if estimateCapped}
                <span class="pill">capped</span>
              {:else if $settings.targetMB > 0}
                <span class="pill">fits {$settings.targetMB} MB</span>
              {/if}
            </span>
          </div>
          <button class="btn primary lg block" type="button" on:click={convert} disabled={!canConvert}>
            Make {formatLabel($settings.format)}
          </button>
          {#if validationMessage}
            <p class="hint">{validationMessage}</p>
          {/if}
        {/if}
      </div>
    </div>

    <!-- RIGHT OPTIONS -->
    <div class="panel">
      <div class="scroll">
        {#if src.kind === "video"}
          <VideoTiming bind:startSec bind:endSec {durationSec} />
        {:else}
          <ImagesTiming />
        {/if}
        <FormatPicker />
        <SizeControls {outHeight} />
        <PlaybackControls />
        <QualityControls />
        <OutputControls />
      </div>
    </div>
  </div>
{/if}

<style>
  .estimate-val {
    display: flex;
    align-items: center;
    gap: 8px;
  }
</style>
