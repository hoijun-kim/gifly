<script lang="ts">
  // Video-specific timing: the trim window (start/end seconds, clamped to the
  // clip) and the output frame rate. FPS lives in the shared settings store;
  // the trim seconds are owned by Workspace and bound in.
  import { settings } from "../lib/settings";
  import { msToTimecode } from "../lib/format";

  export let startSec: number;
  export let endSec: number;
  export let durationSec: number;

  const FPS_PRESETS = [10, 15, 24, 30];

  $: startMs = Math.round(startSec * 1000);
  $: endMs = Math.round(endSec * 1000);

  function clampStart() {
    if (startSec < 0) startSec = 0;
    if (startSec > durationSec) startSec = durationSec;
  }
  function clampEnd() {
    if (endSec < 0) endSec = 0;
    if (endSec > durationSec) endSec = durationSec;
  }
</script>

<section class="section">
  <span class="eyebrow">Timing</span>
  <div class="field-row">
    <label class="field">
      <span class="field-label">Start (s)</span>
      <input class="mono" type="number" min="0" max={durationSec} step="0.1" bind:value={startSec} on:blur={clampStart} />
      <span class="timecode mono">{msToTimecode(startMs)}</span>
    </label>
    <label class="field">
      <span class="field-label">End (s)</span>
      <input class="mono" type="number" min="0" max={durationSec} step="0.1" bind:value={endSec} on:blur={clampEnd} />
      <span class="timecode mono">{msToTimecode(endMs)}</span>
    </label>
  </div>
  <label class="field field-wide">
    <span class="field-label">FPS</span>
    <input class="mono" type="number" min="1" step="1" bind:value={$settings.fps} />
  </label>
  <div class="chip-row" role="group" aria-label="FPS presets">
    {#each FPS_PRESETS as preset (preset)}
      <button type="button" class="chip mono" class:active={Math.round($settings.fps) === preset} on:click={() => ($settings.fps = preset)}>
        {preset}
      </button>
    {/each}
  </div>
</section>
