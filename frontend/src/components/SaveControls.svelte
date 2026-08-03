<script lang="ts">
  // Where and how the output is saved: folder (beside the source or a chosen
  // folder), file name, what to do if the name is taken, and whether to reveal
  // the file when done. These preferences are remembered across restarts.
  import { settings } from "../lib/settings";
  import { source } from "../lib/source";
  import { PickFolder } from "../lib/wails";
  import { formatExt } from "../lib/format";
  import { defaultOutName } from "../lib/request";
  import type { FolderMode, OnExist } from "../lib/settings";

  const FOLDERS: { id: FolderMode; label: string }[] = [
    { id: "source", label: "Beside source" },
    { id: "custom", label: "Choose folder" },
  ];
  const POLICIES: { id: OnExist; label: string }[] = [
    { id: "number", label: "Number" },
    { id: "overwrite", label: "Overwrite" },
    { id: "timestamp", label: "Timestamp" },
  ];
  const POLICY_HINT: Record<OnExist, string> = {
    number: "If the name is taken, add (2), (3)...",
    overwrite: "Replace any file with the same name.",
    timestamp: "Append the date and time to the name.",
  };

  let browsing = false;

  $: name = $settings.outName.trim() || defaultOutName($source);
  $: savesAs = `${name}${formatExt($settings.format)}`;

  function setFolderMode(m: FolderMode) {
    settings.update((s) => ({ ...s, outFolderMode: m }));
    if (m === "custom" && !$settings.outFolder) browse();
  }
  function setPolicy(p: OnExist) {
    settings.update((s) => ({ ...s, onExist: p }));
  }
  async function browse() {
    browsing = true;
    try {
      const dir = await PickFolder();
      settings.update((s) => ({ ...s, outFolder: dir, outFolderMode: "custom" }));
    } catch {
      // dialog cancelled - keep the current folder
    } finally {
      browsing = false;
    }
  }
</script>

<div class="card">
  <div class="card-head"><span class="eyebrow">Save</span><span class="card-note">{savesAs}</span></div>
  <div class="card-body">
    <div class="field">
      <span class="field-label">Folder</span>
      <div class="seg" role="group" aria-label="Output folder">
        {#each FOLDERS as f (f.id)}
          <button type="button" class:on={$settings.outFolderMode === f.id} on:click={() => setFolderMode(f.id)}>{f.label}</button>
        {/each}
      </div>
    </div>
    {#if $settings.outFolderMode === "custom"}
      <div class="folder-row">
        <span class="folder-path mono" title={$settings.outFolder}>{$settings.outFolder || "No folder chosen"}</span>
        <button class="btn ghost sm" type="button" on:click={browse} disabled={browsing}>{browsing ? "..." : "Browse"}</button>
      </div>
    {/if}

    <label class="field">
      <span class="field-label">File name</span>
      <input class="mono" type="text" placeholder={defaultOutName($source)} bind:value={$settings.outName} />
    </label>

    <div class="field">
      <span class="field-label">If a file exists</span>
      <div class="seg" role="group" aria-label="On existing file">
        {#each POLICIES as p (p.id)}
          <button type="button" class:on={$settings.onExist === p.id} on:click={() => setPolicy(p.id)}>{p.label}</button>
        {/each}
      </div>
      <span class="hint">{POLICY_HINT[$settings.onExist]}</span>
    </div>

    <label class="switch">
      <input type="checkbox" bind:checked={$settings.autoReveal} /><span class="track"></span> Reveal in Explorer when done
    </label>
  </div>
</div>

<style>
  .folder-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .folder-path {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding: 8px 10px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: var(--r);
    color: var(--muted);
    font-size: 12px;
  }
</style>
