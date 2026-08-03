<script lang="ts">
  // Output format: GIF / WebP / APNG. Drives which Quality controls show and
  // the output file extension.
  import { settings } from "../lib/settings";
  import type { Format } from "../lib/settings";

  const FORMATS: { id: Format; label: string; hint: string }[] = [
    { id: "gif", label: "GIF", hint: "Universal - plays everywhere, 256 colors, larger files." },
    { id: "webp", label: "WebP", hint: "Modern - much smaller, lossy, wide support." },
    { id: "apng", label: "APNG", hint: "Lossless - sharp, full color, largest files." },
  ];

  $: current = FORMATS.find((f) => f.id === $settings.format) ?? FORMATS[0];
</script>

<section class="section">
  <span class="eyebrow">Format</span>
  <div class="chip-row" role="group" aria-label="Output format">
    {#each FORMATS as f (f.id)}
      <button type="button" class="chip" class:active={$settings.format === f.id} on:click={() => ($settings.format = f.id)}>
        {f.label}
      </button>
    {/each}
  </div>
  <p class="hint">{current.hint}</p>
</section>
