<script lang="ts">
  // The source area: the empty-state pickers (video OR images), and, once a
  // source is chosen, its summary - a video's name/duration/size, or the
  // reorderable image list. Writes the shared `source` store; the rest of the
  // screen reacts to it.
  import { source } from "../lib/source";
  import { PickVideo, PickImages } from "../lib/wails";
  import { msToTimecode } from "../lib/format";

  let picking = false;
  let pickError: string | null = null;
  let thumbErrors: Record<string, boolean> = {};

  $: src = $source;

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }
  // A closed file dialog rejects with context.Canceled - that is a no-op, not
  // an error to show.
  function isCancel(err: unknown): boolean {
    return errorMessage(err).toLowerCase().includes("cancel");
  }
  function basename(path: string): string {
    const idx = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
    return idx >= 0 ? path.slice(idx + 1) : path;
  }
  function extOf(path: string): string {
    const idx = path.lastIndexOf(".");
    return idx >= 0 && idx < path.length - 1 ? path.slice(idx + 1).toUpperCase() : "IMG";
  }
  // Best-effort local thumbnail - some WebView2 configs block file:// fetches
  // from the app origin, so a broken image falls back to an extension badge.
  function fileUrl(path: string): string {
    const normalized = path.replace(/\\/g, "/");
    const prefixed = normalized.startsWith("/") ? normalized : `/${normalized}`;
    return `file://${encodeURI(prefixed)}`;
  }
  function onThumbError(path: string) {
    thumbErrors = { ...thumbErrors, [path]: true };
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

  function moveUp(i: number) {
    source.update((s) => {
      if (!s || s.kind !== "images" || i <= 0) return s;
      const items = s.items.slice();
      [items[i - 1], items[i]] = [items[i], items[i - 1]];
      return { kind: "images", items };
    });
  }
  function moveDown(i: number) {
    source.update((s) => {
      if (!s || s.kind !== "images" || i >= s.items.length - 1) return s;
      const items = s.items.slice();
      [items[i], items[i + 1]] = [items[i + 1], items[i]];
      return { kind: "images", items };
    });
  }
  function removeAt(i: number) {
    source.update((s) => {
      if (!s || s.kind !== "images") return s;
      const items = s.items.filter((_, idx) => idx !== i);
      return items.length ? { kind: "images", items } : null;
    });
  }
  function clearImages() {
    source.set(null);
    thumbErrors = {};
  }
</script>

{#if !src}
  <div class="pick-panel">
    <div class="pick-actions">
      <button class="btn-primary" type="button" on:click={pickVideo} disabled={picking}>
        {picking ? "Opening..." : "Pick a video"}
      </button>
      <button class="btn-ghost" type="button" on:click={pickImages} disabled={picking}>
        {picking ? "Opening..." : "Add images"}
      </button>
    </div>
    <p class="hint">Trim a video, or sequence a set of images - one GIF, WebP or APNG.</p>
    {#if pickError}<p class="error">{pickError}</p>{/if}
  </div>
{:else if src.kind === "video"}
  <section class="section">
    <div class="source-row">
      <span class="eyebrow">Source</span>
      <button class="btn-ghost" type="button" on:click={pickVideo} disabled={picking}>
        {picking ? "Opening..." : "Change"}
      </button>
    </div>
    <span class="source-name" title={src.info.Path}>{basename(src.info.Path)}</span>
    <dl class="meta">
      <div class="meta-item">
        <dt>Duration</dt>
        <dd class="mono">{msToTimecode(src.info.DurationMS)}</dd>
      </div>
      <div class="meta-item">
        <dt>Source size</dt>
        <dd class="mono">{src.info.Width}x{src.info.Height}</dd>
      </div>
    </dl>
    {#if pickError}<p class="error">{pickError}</p>{/if}
  </section>
{:else}
  <section class="section">
    <div class="source-row">
      <span class="eyebrow">Source <span class="mono">({src.items.length})</span></span>
      <div class="btn-row">
        <button class="btn-ghost" type="button" on:click={pickImages} disabled={picking}>
          {picking ? "Opening..." : "Add images"}
        </button>
        <button class="btn-ghost" type="button" on:click={clearImages}>Clear</button>
      </div>
    </div>
    {#if pickError}<p class="error">{pickError}</p>{/if}
    <ol class="list">
      {#each src.items as img, i (img.Path + i)}
        <li class="row">
          <div class="thumb">
            {#if !thumbErrors[img.Path]}
              <img src={fileUrl(img.Path)} alt="" on:error={() => onThumbError(img.Path)} />
            {:else}
              <span class="thumb-fallback mono">{extOf(img.Path)}</span>
            {/if}
          </div>
          <div class="row-info">
            <span class="row-name" title={img.Path}>{basename(img.Path)}</span>
            <span class="row-dims mono">{img.Width}x{img.Height}</span>
          </div>
          <div class="row-actions">
            <button class="btn-icon" type="button" on:click={() => moveUp(i)} disabled={i === 0} aria-label="Move up">Up</button>
            <button class="btn-icon" type="button" on:click={() => moveDown(i)} disabled={i === src.items.length - 1} aria-label="Move down">Down</button>
            <button class="btn-icon" type="button" on:click={() => removeAt(i)} aria-label="Remove">Remove</button>
          </div>
        </li>
      {/each}
    </ol>
  </section>
{/if}

<style>
  .pick-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: center;
  }
  .list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: 260px;
    overflow-y: auto;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: var(--r);
  }
  .thumb {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    width: 40px;
    height: 40px;
    overflow: hidden;
    background: var(--bg);
    border: 1px solid var(--line);
    border-radius: 6px;
  }
  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .thumb-fallback {
    font-size: 10px;
    color: var(--faint);
  }
  .row-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1 1 auto;
    min-width: 0;
  }
  .row-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
    color: var(--text);
  }
  .row-dims {
    font-size: 11px;
    color: var(--faint);
  }
  .row-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 4px;
  }
  .btn-icon {
    padding: 5px 8px;
    border: 1px solid var(--line);
    border-radius: var(--r);
    background: transparent;
    color: var(--text);
    font-family: var(--ui);
    font-size: 11px;
    cursor: pointer;
  }
  .btn-icon:hover:not(:disabled) {
    background: var(--panel);
  }
  .btn-icon:disabled {
    color: var(--faint);
    cursor: not-allowed;
    opacity: 0.5;
  }
  @media (prefers-reduced-motion: no-preference) {
    .btn-icon {
      transition: background-color 120ms ease;
    }
  }
</style>
