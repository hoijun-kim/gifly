<script lang="ts">
  // The left-stage source area: the empty-state drop zone (pick a video OR add
  // images), and once a source is chosen a compact summary row. The image
  // reorder list itself lives in the right panel (ImagesTiming).
  import { source } from "../lib/source";
  import { PickVideo, PickImages } from "../lib/wails";
  import { msToTimecode } from "../lib/format";

  let picking = false;
  let pickError: string | null = null;

  $: src = $source;

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }
  // A closed file dialog rejects with context.Canceled - a no-op, not an error.
  function isCancel(err: unknown): boolean {
    return errorMessage(err).toLowerCase().includes("cancel");
  }
  function basename(path: string): string {
    const idx = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
    return idx >= 0 ? path.slice(idx + 1) : path;
  }

  async function pickVideo() {
    pickError = null;
    picking = true;
    try {
      const info = await PickVideo();
      source.set({ kind: "video", info });
    } catch (err) {
      if (!isCancel(err)) pickError = errorMessage(err);
    } finally {
      picking = false;
    }
  }

  async function pickImages() {
    pickError = null;
    picking = true;
    try {
      const picked = await PickImages();
      source.update((s) => {
        const existing = s && s.kind === "images" ? s.items : [];
        return { kind: "images", items: [...existing, ...picked] };
      });
    } catch (err) {
      if (!isCancel(err)) pickError = errorMessage(err);
    } finally {
      picking = false;
    }
  }

  function clearSource() {
    source.set(null);
  }
</script>

{#if !src}
  <div class="dropzone">
    <div class="glyph">&#127916;</div>
    <p>Drop a <b>video</b> to trim, or a set of <b>images</b> to sequence - one GIF, WebP or APNG.</p>
    <div class="dz-actions">
      <button class="btn primary" type="button" on:click={pickVideo} disabled={picking}>
        {picking ? "Opening..." : "Pick a video"}
      </button>
      <button class="btn ghost" type="button" on:click={pickImages} disabled={picking}>
        {picking ? "Opening..." : "Add images"}
      </button>
    </div>
    {#if pickError}<p class="error">{pickError}</p>{/if}
  </div>
{:else if src.kind === "video"}
  <div class="source-row">
    <div class="source-thumb"></div>
    <div class="source-info">
      <span class="source-name" title={src.info.Path}>{basename(src.info.Path)}</span>
      <span class="source-sub mono">{src.info.Width}x{src.info.Height} &middot; {msToTimecode(src.info.DurationMS)}</span>
    </div>
    <div class="source-actions">
      <button class="link" type="button" on:click={pickVideo} disabled={picking}>{picking ? "..." : "Change"}</button>
    </div>
  </div>
  {#if pickError}<p class="error">{pickError}</p>{/if}
{:else}
  <div class="source-row">
    <div class="source-thumb"></div>
    <div class="source-info">
      <span class="source-name">{src.items.length} image{src.items.length === 1 ? "" : "s"}</span>
      <span class="source-sub">ordered in the frames panel</span>
    </div>
    <div class="source-actions">
      <button class="link" type="button" on:click={clearSource}>Clear</button>
    </div>
  </div>
  {#if pickError}<p class="error">{pickError}</p>{/if}
{/if}
