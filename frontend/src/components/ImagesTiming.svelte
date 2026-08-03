<script lang="ts">
  // Images-specific timing: how long each frame shows (ms), with a derived fps
  // readout and the same preset chips as the video path. Frame duration lives
  // in the shared settings store.
  import { settings } from "../lib/settings";
  import { fpsFromFrameMs, frameMsFromFps } from "../lib/format";

  const FPS_PRESETS = [10, 15, 24, 30];

  $: fps = fpsFromFrameMs($settings.frameMs);
</script>

<section class="section">
  <span class="eyebrow">Timing</span>
  <div class="field-row">
    <label class="field">
      <span class="field-label">Frame duration (ms)</span>
      <input class="mono" type="number" min="1" step="1" bind:value={$settings.frameMs} />
    </label>
    <div class="field">
      <span class="field-label">Rate</span>
      <span class="mono readout">= {fps.toFixed(1)} fps</span>
    </div>
  </div>
  <div class="chip-row" role="group" aria-label="FPS presets">
    {#each FPS_PRESETS as preset (preset)}
      <button type="button" class="chip mono" class:active={Math.round(fps) === preset} on:click={() => ($settings.frameMs = Math.round(frameMsFromFps(preset)))}>
        {preset}
      </button>
    {/each}
  </div>
</section>
