<script lang="ts">
  import { onMount } from "svelte";
  import "./style.css";
  import TitleBar from "./components/TitleBar.svelte";
  import Workspace from "./components/Workspace.svelte";
  import { initPersistence } from "./lib/persist";
  import { initTheme } from "./lib/theme";

  onMount(() => {
    initTheme();
    initPersistence();
  });
</script>

<div id="app">
  <TitleBar />
  <Workspace />
</div>
