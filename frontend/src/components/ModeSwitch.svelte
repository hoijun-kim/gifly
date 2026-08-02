<script lang="ts">
  // A segmented Video|Images toggle bound to the shared `mode` store
  // (frontend/src/lib/wails.ts). App.svelte reads the same store to decide
  // which mode component to mount.
  import { mode, type Mode } from "../lib/wails";

  const options: { value: Mode; label: string }[] = [
    { value: "video", label: "Video" },
    { value: "images", label: "Images" },
  ];
</script>

<div class="mode-switch" role="tablist" aria-label="Conversion mode">
  {#each options as opt (opt.value)}
    <button
      type="button"
      role="tab"
      class="mode-btn"
      class:active={$mode === opt.value}
      aria-selected={$mode === opt.value}
      on:click={() => mode.set(opt.value)}
    >
      {opt.label}
    </button>
  {/each}
</div>

<style>
  .mode-switch {
    display: flex;
    gap: 2px;
    padding: 3px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: var(--r);
  }

  .mode-btn {
    flex: 1 1 0;
    padding: 7px 0;
    border: none;
    border-radius: calc(var(--r) - 3px);
    background: transparent;
    color: var(--muted);
    font-family: var(--ui);
    font-size: 13px;
    font-weight: 550;
    cursor: pointer;
  }

  .mode-btn:hover:not(.active) {
    color: var(--text);
  }

  .mode-btn.active {
    background: var(--accent);
    color: var(--accent-ink);
  }

  @media (prefers-reduced-motion: no-preference) {
    .mode-btn {
      transition: background-color 150ms ease, color 150ms ease;
    }
  }
</style>
