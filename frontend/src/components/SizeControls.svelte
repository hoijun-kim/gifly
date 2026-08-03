<script lang="ts">
  // Crop/aspect + output width. The height is derived (Workspace passes it in).
  // Changing width or aspect manually clears any active platform preset.
  import { settings } from "../lib/settings";
  import type { Aspect } from "../lib/settings";

  export let outHeight: number;

  const ASPECTS: { id: Aspect; label: string }[] = [
    { id: "", label: "Free" },
    { id: "1:1", label: "1:1" },
    { id: "16:9", label: "16:9" },
    { id: "9:16", label: "9:16" },
  ];
  const WIDTH_PRESETS = [240, 360, 480, 640];

  function setAspect(a: Aspect) {
    settings.update((s) => ({ ...s, aspect: a }));
  }
  function pickWidth(w: number) {
    settings.update((s) => ({ ...s, width: w, preset: "" }));
  }
  function onWidthInput() {
    settings.update((s) => ({ ...s, preset: "" }));
  }
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Size</span><span class="card-note">{Math.round($settings.width)} &times; {outHeight}</span></div>
  <div class="card-body">
    <div class="seg" role="group" aria-label="Aspect ratio">
      {#each ASPECTS as a (a.id)}
        <button type="button" class:on={$settings.aspect === a.id} on:click={() => setAspect(a.id)}>{a.label}</button>
      {/each}
    </div>
    <div class="row2">
      <label class="field">
        <span class="field-label">Width (px)</span>
        <input class="mono" type="number" min="2" step="1" bind:value={$settings.width} on:input={onWidthInput} />
      </label>
      <div class="field">
        <span class="field-label">Height</span>
        <span class="readout">{outHeight} px</span>
      </div>
    </div>
    <div class="chips" role="group" aria-label="Width presets">
      {#each WIDTH_PRESETS as preset (preset)}
        <button type="button" class="chip" class:on={Math.round($settings.width) === preset} on:click={() => pickWidth(preset)}>
          {preset}
        </button>
      {/each}
    </div>
  </div>
</div>
