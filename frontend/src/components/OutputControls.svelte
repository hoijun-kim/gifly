<script lang="ts">
  // Platform presets (which set a max width + size target), an explicit target
  // size, and the live size estimate (Workspace computes and passes it in).
  import { settings } from "../lib/settings";
  import { PLATFORM_PRESETS } from "../lib/request";
  import { humanBytes } from "../lib/format";

  export let displayEstimate: number;
  export let estimateCapped = false;

  const PRESET_KEYS = Object.keys(PLATFORM_PRESETS);

  function applyPreset(key: string) {
    const p = PLATFORM_PRESETS[key];
    settings.update((s) => ({ ...s, preset: key, width: p.maxWidth, targetMB: p.targetMB }));
  }
  function clearPreset() {
    settings.update((s) => ({ ...s, preset: "" }));
  }
</script>

<section class="section">
  <span class="eyebrow">Output</span>
  <div class="chip-row" role="group" aria-label="Platform preset">
    <button type="button" class="chip" class:active={$settings.preset === ""} on:click={clearPreset}>None</button>
    {#each PRESET_KEYS as key (key)}
      <button type="button" class="chip" class:active={$settings.preset === key} on:click={() => applyPreset(key)}>
        {PLATFORM_PRESETS[key].label}
      </button>
    {/each}
  </div>
  <label class="field field-wide">
    <span class="field-label">Target size (MB) <span class="field-optional">optional</span></span>
    <input class="mono" type="number" min="0" step="0.1" placeholder="off" bind:value={$settings.targetMB} on:input={clearPreset} />
  </label>
  {#if $settings.targetMB > 0}
    <p class="hint">gifly shrinks the width to fit ~{$settings.targetMB} MB</p>
  {/if}
  <p class="estimate">
    <span>Estimated size</span>
    <span class="estimate-value mono">~{humanBytes(displayEstimate)}</span>
    {#if estimateCapped}<span class="estimate-note">(capped by target)</span>{/if}
  </p>
</section>
