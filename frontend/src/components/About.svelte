<script lang="ts">
  // The About dialog: app name/version, author, links, and license. Opened from
  // the title bar's info button.
  import { BrowserOpenURL } from "../lib/wails";

  export let open = false;

  const VERSION = "0.1.0";

  function close() {
    open = false;
  }
  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape") close();
  }
  function openURL(url: string) {
    BrowserOpenURL(url);
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions a11y-no-noninteractive-element-interactions -->
  <div class="backdrop" role="presentation" on:click={close}>
    <div class="dialog" role="dialog" aria-modal="true" aria-label="About gifly" on:click|stopPropagation>
      <button class="x" type="button" on:click={close} aria-label="Close">&#10005;</button>
      <div class="head">
        <svg class="mark" viewBox="0 0 32 32" aria-hidden="true">
          <rect x="4" y="11" width="17" height="17" rx="4.2" fill="none" stroke="var(--accent-hi)" stroke-width="2.6" />
          <rect x="11" y="4" width="17" height="17" rx="4.2" fill="var(--accent)" />
        </svg>
        <div>
          <div class="name">gifly <span class="ver mono">v{VERSION}</span></div>
          <div class="tag">video &amp; images to GIF, WebP or APNG</div>
        </div>
      </div>

      <div class="by">Made by <b>H.K</b></div>

      <div class="links">
        <button class="link" type="button" on:click={() => openURL("https://github.com/hoijun-kim/gifly")}>GitHub</button>
        <button class="link" type="button" on:click={() => openURL("https://hoijun-kim.github.io/gifly/")}>Website</button>
      </div>

      <div class="legal">
        <p>Licensed under the <button class="ln" type="button" on:click={() => openURL("https://github.com/hoijun-kim/gifly/blob/master/LICENSE")}>PolyForm Noncommercial 1.0.0</button>.</p>
        <p>Bundles <button class="ln" type="button" on:click={() => openURL("https://ffmpeg.org")}>FFmpeg</button>, used under the LGPL-3.0.</p>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgba(0, 0, 0, 0.5);
  }
  .dialog {
    position: relative;
    width: min(360px, 100%);
    border: 1px solid var(--line);
    border-radius: var(--r-lg);
    background: var(--panel);
    box-shadow: var(--shadow-panel);
    padding: 22px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .x {
    position: absolute;
    top: 10px;
    right: 10px;
    width: 28px;
    height: 28px;
    border: 0;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--muted);
    cursor: pointer;
    font-size: 12px;
  }
  .x:hover {
    background: var(--panel-2);
    color: var(--text);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 13px;
  }
  .mark {
    width: 40px;
    height: 40px;
    flex: 0 0 auto;
  }
  .name {
    font-size: 19px;
    font-weight: 750;
    letter-spacing: -0.01em;
  }
  .ver {
    font-size: 12px;
    font-weight: 600;
    color: var(--faint);
  }
  .tag {
    font-size: 12px;
    color: var(--muted);
  }
  .by {
    font-size: 14px;
    color: var(--muted);
  }
  .by b {
    color: var(--text);
  }
  .links {
    display: flex;
    gap: 8px;
  }
  .link {
    padding: 8px 14px;
    border: 1px solid var(--line);
    border-radius: var(--r);
    background: var(--panel-2);
    color: var(--text);
    font-family: var(--ui);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }
  .link:hover {
    border-color: var(--faint);
    background: var(--panel-3);
  }
  .legal {
    border-top: 1px solid var(--line);
    padding-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .legal p {
    margin: 0;
    font-size: 12px;
    color: var(--faint);
    line-height: 1.5;
  }
  .ln {
    background: 0;
    border: 0;
    padding: 0;
    font: inherit;
    color: var(--accent-hi);
    cursor: pointer;
  }
  .ln:hover {
    text-decoration: underline;
  }
</style>
