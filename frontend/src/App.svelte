<script lang="ts">
  import "./style.css";
  import TitleBar from "./components/TitleBar.svelte";
  import Workspace from "./components/Workspace.svelte";
</script>

<div id="app">
  <TitleBar />
  <main class="work">
    <Workspace />
  </main>
</div>
