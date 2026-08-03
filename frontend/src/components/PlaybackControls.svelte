<script lang="ts">
  // Shared playback options: loop count (segmented, no native dropdown),
  // playback speed, reverse and boomerang.
  import { settings } from "../lib/settings";
  import type { LoopChoice } from "../lib/settings";

  const LOOPS: { id: LoopChoice; label: string }[] = [
    { id: "forever", label: "Forever" },
    { id: "once", label: "Once" },
    { id: "custom", label: "Custom" },
  ];

  function setLoop(c: LoopChoice) {
    settings.update((s) => ({ ...s, loopChoice: c }));
  }
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Playback</span></div>
  <div class="card-body">
    <div class="field">
      <span class="field-label">Repeat</span>
      <div class="seg" role="group" aria-label="Repeat">
        {#each LOOPS as l (l.id)}
          <button type="button" class:on={$settings.loopChoice === l.id} on:click={() => setLoop(l.id)}>{l.label}</button>
        {/each}
      </div>
    </div>
    {#if $settings.loopChoice === "custom"}
      <label class="field">
        <span class="field-label">Times</span>
        <input class="mono" type="number" min="1" step="1" bind:value={$settings.loopCount} />
      </label>
    {/if}
    <div class="field">
      <span class="field-label">Speed <span class="val">{$settings.speed.toFixed(2)}x</span></span>
      <input type="range" min="0.25" max="4" step="0.05" bind:value={$settings.speed} />
    </div>
    <div class="switch-row">
      <label class="switch"><input type="checkbox" bind:checked={$settings.reverse} /><span class="track"></span> Reverse</label>
      <label class="switch"><input type="checkbox" bind:checked={$settings.boomerang} /><span class="track"></span> Boomerang</label>
    </div>
  </div>
</div>
