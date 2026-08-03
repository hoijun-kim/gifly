<script lang="ts">
  // Output format: GIF / WebP / APNG. Drives which Quality controls show and
  // the output file extension.
  import { settings } from "../lib/settings";
  import { formatExt } from "../lib/format";
  import type { Format } from "../lib/settings";

  const FORMATS: { id: Format; label: string; hint: string }[] = [
    { id: "gif", label: "GIF", hint: "Universal - plays everywhere, 256 colors, larger files." },
    { id: "webp", label: "WebP", hint: "Modern - much smaller, lossy, wide support." },
    { id: "apng", label: "APNG", hint: "Lossless - sharp, full color, largest files." },
  ];

  $: current = FORMATS.find((f) => f.id === $settings.format) ?? FORMATS[0];
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Format</span><span class="card-note">{formatExt($settings.format)}</span></div>
  <div class="card-body">
    <div class="seg" role="group" aria-label="Output format">
      {#each FORMATS as f (f.id)}
        <button type="button" class:on={$settings.format === f.id} on:click={() => ($settings.format = f.id)}>{f.label}</button>
      {/each}
    </div>
    <p class="hint">{current.hint}</p>
  </div>
</div>
