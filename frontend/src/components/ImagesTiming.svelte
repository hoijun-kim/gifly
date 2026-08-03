<script lang="ts">
  // Images source + timing (right panel): the reorderable frame list, an add
  // button, and how long each frame shows (with a derived fps readout). Frame
  // duration lives in the shared settings store; the ordered list lives in the
  // source store.
  import { source } from "../lib/source";
  import { settings } from "../lib/settings";
  import { PickImages } from "../lib/wails";
  import { fpsFromFrameMs, frameMsFromFps } from "../lib/format";

  const FPS_PRESETS = [10, 15, 24, 30];

  let picking = false;
  let pickError: string | null = null;
  let thumbErrors: Record<string, boolean> = {};

  $: src = $source;
  $: fps = fpsFromFrameMs($settings.frameMs);

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }
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

  async function addImages() {
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
</script>

<div class="card">
  <div class="card-head">
    <span class="eyebrow">Frames</span>
    {#if src && src.kind === "images"}<span class="card-note">{src.items.length} &middot; drag order</span>{/if}
  </div>
  <div class="card-body">
    {#if src && src.kind === "images"}
      <ol class="imglist">
        {#each src.items as img, i (img.Path + i)}
          <li class="imgrow">
            <div class="imgthumb">
              {#if !thumbErrors[img.Path]}
                <img src={fileUrl(img.Path)} alt="" on:error={() => onThumbError(img.Path)} />
              {:else}
                <span class="imgthumb-fallback">{extOf(img.Path)}</span>
              {/if}
            </div>
            <div class="imgrow-info">
              <span class="imgrow-name" title={img.Path}>{basename(img.Path)}</span>
              <span class="imgrow-dims">{img.Width}x{img.Height}</span>
            </div>
            <div class="imgrow-actions">
              <button class="btn-icon" type="button" on:click={() => moveUp(i)} disabled={i === 0} aria-label="Move up">Up</button>
              <button class="btn-icon" type="button" on:click={() => moveDown(i)} disabled={i === src.items.length - 1} aria-label="Move down">Down</button>
              <button class="btn-icon" type="button" on:click={() => removeAt(i)} aria-label="Remove">&#10005;</button>
            </div>
          </li>
        {/each}
      </ol>
    {/if}
    <button class="btn ghost sm" type="button" on:click={addImages} disabled={picking}>
      {picking ? "Opening..." : "+ Add images"}
    </button>
    {#if pickError}<p class="error">{pickError}</p>{/if}

    <div class="row2">
      <label class="field">
        <span class="field-label">Frame duration (ms)</span>
        <input class="mono" type="number" min="1" step="1" bind:value={$settings.frameMs} />
      </label>
      <div class="field">
        <span class="field-label">Rate</span>
        <span class="readout">= {fps.toFixed(1)} fps</span>
      </div>
    </div>
    <div class="chips" role="group" aria-label="FPS presets">
      {#each FPS_PRESETS as preset (preset)}
        <button type="button" class="chip" class:on={Math.round(fps) === preset} on:click={() => ($settings.frameMs = Math.round(frameMsFromFps(preset)))}>
          {preset}
        </button>
      {/each}
    </div>
  </div>
</div>
