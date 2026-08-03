<script lang="ts">
  // Format-conditional quality: GIF gets a color count + dither method, WebP a
  // quality slider, APNG nothing (lossless).
  import { settings } from "../lib/settings";
  import type { DitherMethod } from "../lib/settings";

  const DITHERS: { id: DitherMethod; label: string }[] = [
    { id: "none", label: "None" },
    { id: "bayer", label: "Bayer" },
    { id: "sierra2", label: "Sierra2" },
    { id: "floyd", label: "Floyd" },
  ];

  function setDither(d: DitherMethod) {
    settings.update((s) => ({ ...s, dither: d }));
  }
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Quality</span></div>
  <div class="card-body">
    {#if $settings.format === "gif"}
      <div class="field">
        <span class="field-label">Colors <span class="val">{$settings.colors}</span></span>
        <input type="range" min="2" max="256" step="1" bind:value={$settings.colors} />
      </div>
      <div class="field">
        <span class="field-label">Dither</span>
        <div class="seg" role="group" aria-label="Dither method">
          {#each DITHERS as d (d.id)}
            <button type="button" class:on={$settings.dither === d.id} on:click={() => setDither(d.id)}>{d.label}</button>
          {/each}
        </div>
      </div>
    {:else if $settings.format === "webp"}
      <div class="field">
        <span class="field-label">Quality <span class="val">{$settings.webpQuality}</span></span>
        <input type="range" min="0" max="100" step="1" bind:value={$settings.webpQuality} />
      </div>
    {:else}
      <p class="hint">APNG is lossless - no quality options.</p>
    {/if}
  </div>
</div>
