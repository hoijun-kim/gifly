<script lang="ts">
  import "./style.css";
  import TitleBar from "./components/TitleBar.svelte";
  import ModeSwitch from "./components/ModeSwitch.svelte";
  import Video from "./modes/Video.svelte";
  import { mode } from "./lib/wails";

  // Images.svelte (Task 7) replaces the remaining placeholder; the mode
  // store and the layout around it are already the real shape.
</script>

<div id="app">
  <TitleBar />
  <main class="work">
    <ModeSwitch />
    <section class="stage">
      {#if $mode === "video"}
        <Video />
      {:else}
        <div class="placeholder">
          <p>Add images and turn them into an animated GIF.</p>
          <p class="placeholder-note">Images mode arrives in Task 7.</p>
        </div>
      {/if}
    </section>
  </main>
</div>

<style>
  .stage {
    flex: 1 1 auto;
    display: flex;
    min-height: 0;
  }

  .placeholder {
    flex: 1 1 auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 48px 24px;
    border: 1px dashed var(--line);
    border-radius: var(--r-panel);
    background: var(--panel);
    color: var(--muted);
    text-align: center;
  }

  .placeholder p {
    margin: 0;
  }

  .placeholder-note {
    color: var(--faint);
    font-size: 13px;
  }

  @media (prefers-reduced-motion: no-preference) {
    .stage {
      animation: stage-in 200ms ease-out;
    }
  }

  @keyframes stage-in {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
