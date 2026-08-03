<script lang="ts">
  // The single unified screen: pick a video OR a set of images (SourcePanel),
  // set the source-specific timing (VideoTiming / ImagesTiming) and the shared
  // output options (added by later components), then convert and land on the
  // result card. All the pure logic (request building, validation, estimate)
  // lives in ../lib; this orchestrator wires the source/settings stores to the
  // bound Go methods in ../lib/wails.
  import { onDestroy } from "svelte";
  import { source } from "../lib/source";
  import { settings } from "../lib/settings";
  import { ConvertVideo, ConvertImages, Cancel, RevealOutput, onProgress } from "../lib/wails";
  import type { ConvertResult } from "../lib/wails";
  import { humanBytes, formatLabel } from "../lib/format";
  import { videoValid, imagesValid, settingsValid } from "../lib/validate";
  import { buildVideoRequest, buildImagesRequest } from "../lib/request";
  import SourcePanel from "./SourcePanel.svelte";
  import VideoTiming from "./VideoTiming.svelte";
  import ImagesTiming from "./ImagesTiming.svelte";

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

  // Back to the empty state for a fresh source; output options are intentionally
  // kept so a user tuning settings across conversions is not surprised.
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

<div class="workspace">
  <SourcePanel />

  {#if src}
    {#if src.kind === "video"}
      <VideoTiming bind:startSec bind:endSec {durationSec} />
    {:else}
      <ImagesTiming />
    {/if}

    <div class="actions">
      <button class="btn-primary" type="button" on:click={convert} disabled={!canConvert}>
        {converting ? "Converting..." : `Make ${formatLabel($settings.format)}`}
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
        <img class="preview" src={previewUrl} alt="Converted preview" />
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
  .workspace {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
</style>
