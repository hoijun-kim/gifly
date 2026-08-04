<script lang="ts">
  // gifly's window is built Frameless (see main.go), so this bar is the only
  // title bar there is: a drag region plus its own minimize/close, backed by
  // the Wails JS runtime directly (WindowMinimise/Quit) rather than a Go
  // round-trip - there is no in-flight state here that needs to veto a close.
  import { Quit, WindowMinimise } from "../lib/wails";
  import { theme, toggleTheme } from "../lib/theme";
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
    <button class="ctrl-btn" type="button" on:click={toggleTheme} aria-label="Toggle light or dark theme" title="Toggle light/dark">
      {#if $theme === "dark"}
        <svg viewBox="0 0 12 12" aria-hidden="true">
          <circle cx="6" cy="6" r="2.4" fill="none" stroke="currentColor" stroke-width="1.1" />
          <g stroke="currentColor" stroke-width="1.1" stroke-linecap="round">
            <line x1="6" y1="0.8" x2="6" y2="2" /><line x1="6" y1="10" x2="6" y2="11.2" />
            <line x1="0.8" y1="6" x2="2" y2="6" /><line x1="10" y1="6" x2="11.2" y2="6" />
            <line x1="2.3" y1="2.3" x2="3.1" y2="3.1" /><line x1="8.9" y1="8.9" x2="9.7" y2="9.7" />
            <line x1="9.7" y1="2.3" x2="8.9" y2="3.1" /><line x1="3.1" y1="8.9" x2="2.3" y2="9.7" />
          </g>
        </svg>
      {:else}
        <svg viewBox="0 0 12 12" aria-hidden="true">
          <path d="M9.3 7.4A4 4 0 1 1 4.6 2.7 3.2 3.2 0 0 0 9.3 7.4Z" fill="currentColor" />
        </svg>
      {/if}
    </button>
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
