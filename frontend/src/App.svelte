<script lang="ts">
  import "./style.css";
  import TitleBar from "./components/TitleBar.svelte";
  import ModeSwitch from "./components/ModeSwitch.svelte";
  import Video from "./modes/Video.svelte";
  import Images from "./modes/Images.svelte";
  import { mode } from "./lib/wails";
</script>

<div id="app">
  <TitleBar />
  <main class="work">
    <ModeSwitch />
    <section class="stage">
      {#if $mode === "video"}
        <Video />
      {:else}
        <Images />
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
