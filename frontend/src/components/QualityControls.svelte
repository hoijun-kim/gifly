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

<section class="section">
  <span class="eyebrow">Quality</span>
  {#if $settings.format === "gif"}
    <label class="field field-wide">
      <span class="field-label">Colors <span class="mono">{$settings.colors}</span></span>
      <input type="range" min="2" max="256" step="1" bind:value={$settings.colors} />
    </label>
    <div class="field field-wide">
      <span class="field-label">Dither</span>
      <div class="chip-row" role="group" aria-label="Dither method">
        {#each DITHERS as d (d.id)}
          <button type="button" class="chip" class:active={$settings.dither === d.id} on:click={() => setDither(d.id)}>
            {d.label}
          </button>
        {/each}
      </div>
    </div>
  {:else if $settings.format === "webp"}
    <label class="field field-wide">
      <span class="field-label">Quality <span class="mono">{$settings.webpQuality}</span></span>
      <input type="range" min="0" max="100" step="1" bind:value={$settings.webpQuality} />
    </label>
  {:else}
    <p class="hint">APNG is lossless - no quality options.</p>
  {/if}
</section>
