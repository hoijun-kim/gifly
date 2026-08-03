<script lang="ts">
  // Platform presets (which set a max width + size target) and an explicit
  // target size. The live estimate is shown in the left stage, next to the CTA.
  import { settings } from "../lib/settings";
  import { PLATFORM_PRESETS } from "../lib/request";

  const PRESET_KEYS = Object.keys(PLATFORM_PRESETS);

  function applyPreset(key: string) {
    const p = PLATFORM_PRESETS[key];
    settings.update((s) => ({ ...s, preset: key, width: p.maxWidth, targetMB: p.targetMB }));
  }
  function clearPreset() {
    settings.update((s) => ({ ...s, preset: "" }));
  }
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Output</span><span class="card-note">preset &middot; target</span></div>
  <div class="card-body">
    <div class="chips" role="group" aria-label="Platform preset">
      <button type="button" class="chip" class:on={$settings.preset === ""} on:click={clearPreset}>None</button>
      {#each PRESET_KEYS as key (key)}
        <button type="button" class="chip" class:on={$settings.preset === key} on:click={() => applyPreset(key)}>{PLATFORM_PRESETS[key].label}</button>
      {/each}
    </div>
    <label class="field">
      <span class="field-label">Target size (MB) <span class="timecode">optional</span></span>
      <input class="mono" type="number" min="0" step="0.1" placeholder="off" bind:value={$settings.targetMB} on:input={clearPreset} />
    </label>
    {#if $settings.targetMB > 0}
      <p class="hint">gifly shrinks the width to fit ~{$settings.targetMB} MB.</p>
    {/if}
  </div>
</div>
