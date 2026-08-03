<script lang="ts">
  // Shared playback options: loop count, playback speed, reverse and boomerang.
  import { settings } from "../lib/settings";
</script>

<section class="section">
  <span class="eyebrow">Playback</span>
  <div class="field-row">
    <label class="field">
      <span class="field-label">Repeat</span>
      <select bind:value={$settings.loopChoice}>
        <option value="forever">Forever</option>
        <option value="once">Once</option>
        <option value="custom">Custom</option>
      </select>
    </label>
    {#if $settings.loopChoice === "custom"}
      <label class="field">
        <span class="field-label">Times</span>
        <input class="mono" type="number" min="1" step="1" bind:value={$settings.loopCount} />
      </label>
    {/if}
  </div>
  <label class="field field-wide">
    <span class="field-label">Speed <span class="mono">{$settings.speed.toFixed(2)}x</span></span>
    <input type="range" min="0.25" max="4" step="0.05" bind:value={$settings.speed} />
  </label>
  <div class="toggle-row">
    <label class="toggle">
      <input type="checkbox" bind:checked={$settings.reverse} />
      <span>Reverse</span>
    </label>
    <label class="toggle">
      <input type="checkbox" bind:checked={$settings.boomerang} />
      <span>Boomerang</span>
    </label>
  </div>
</section>

<style>
  .toggle-row {
    display: flex;
    gap: 20px;
    flex-wrap: wrap;
  }
</style>
