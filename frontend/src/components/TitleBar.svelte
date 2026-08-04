<script lang="ts">
  // gifly's window is built Frameless (see main.go), so this bar is the only
  // title bar there is: a drag region plus its own minimize/close, backed by
  // the Wails JS runtime directly (WindowMinimise/Quit) rather than a Go
  // round-trip - there is no in-flight state here that needs to veto a close.
  import { Quit, WindowMinimise } from "../lib/wails";
  import About from "./About.svelte";

  let showAbout = false;
</script>

<header class="titlebar" style="--wails-draggable:drag">
  <span class="wordmark">
    <svg class="mark" viewBox="0 0 20 20" aria-hidden="true">
      <rect x="2" y="6" width="11" height="11" rx="2.5" fill="none" stroke="currentColor" stroke-width="1.6" />
      <rect x="7" y="2.5" width="11" height="11" rx="2.5" fill="var(--accent)" opacity="0.9" />
    </svg>
    <span class="wordmark-text">gifly</span>
  </span>
  <div class="controls" style="--wails-draggable:no-drag">
    <button class="ctrl-btn" type="button" on:click={() => (showAbout = true)} aria-label="About gifly" title="About gifly">
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <circle cx="6" cy="6" r="5" fill="none" stroke="currentColor" stroke-width="1.1" />
        <line x1="6" y1="5.3" x2="6" y2="8.6" stroke="currentColor" stroke-width="1.2" />
        <circle cx="6" cy="3.4" r="0.75" fill="currentColor" />
      </svg>
    </button>
    <button class="ctrl-btn" type="button" on:click={() => WindowMinimise()} aria-label="Minimize" title="Minimize">
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <line x1="2.5" y1="6" x2="9.5" y2="6" stroke="currentColor" stroke-width="1.2" />
      </svg>
    </button>
    <button class="ctrl-btn ctrl-btn-close" type="button" on:click={() => Quit()} aria-label="Close" title="Close">
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <line x1="3" y1="3" x2="9" y2="9" stroke="currentColor" stroke-width="1.2" />
        <line x1="9" y1="3" x2="3" y2="9" stroke="currentColor" stroke-width="1.2" />
      </svg>
    </button>
  </div>
</header>

<About bind:open={showAbout} />

<style>
  .titlebar {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 38px;
    padding: 0 0 0 14px;
    background: var(--panel);
    border-bottom: 1px solid var(--line);
    color: var(--text);
    user-select: none;
  }

  .wordmark {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .mark {
    width: 15px;
    height: 15px;
    display: block;
    color: var(--accent-hi);
  }

  .wordmark-text {
    font-weight: 600;
    font-size: 14px;
    letter-spacing: 0.02em;
    color: var(--text);
  }

  .controls {
    display: flex;
    height: 100%;
  }

  .ctrl-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 100%;
    border: none;
    border-radius: 0;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
  }

  .ctrl-btn:hover {
    background: var(--panel-2);
    color: var(--text);
  }

  .ctrl-btn svg {
    width: 12px;
    height: 12px;
  }

  .ctrl-btn-close:hover {
    background: var(--stop);
    color: #fff;
  }

  @media (prefers-reduced-motion: no-preference) {
    .ctrl-btn {
      transition: background-color 120ms ease, color 120ms ease;
    }
  }
</style>
