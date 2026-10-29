<script lang="ts">
  import { Sites, Layout, SaveLayout, HomeDir } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteInfo, HomeItem } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import { t } from "./i18n";

  // The list of sites, organized by the user: drag a site onto another to
  // make a folder, onto a folder to put it inside, onto a card's edge to
  // change the order. The "⋯" menu does the same without dragging.
  let { onOpen, onNew, onEmpty }: { onOpen: (s: SiteInfo) => void; onNew: () => void; onEmpty: () => void } = $props();
  const h = t.home;

  let sites = $state<Record<string, SiteInfo>>({});
  let items = $state<HomeItem[]>([]);
  let loaded = $state(false);
  let error = $state("");
  let open = $state<string | null>(null); // the open folder's name
  let menu = $state(""); // the site whose "⋯" menu is open
  let renaming = $state("");

  Promise.all([Sites(), Layout()])
    .then(([list, layout]) => {
      for (const s of list ?? []) sites[s.dir] = s;
      items = layout ?? [];
      if (!(list ?? []).length) onEmpty();
    })
    .catch((e) => (error = String(e)))
    .finally(() => (loaded = true));

  const save = () => SaveLayout($state.snapshot(items) as HomeItem[]).catch((e) => (error = String(e)));
  const folderAt = (name: string) => items.findIndex((i) => i.folder === name);
  const openItems = $derived(open === null ? [] : (items[folderAt(open)]?.sites ?? []));

  // A new folder name that isn't taken yet.
  function newName() {
    let name = h.newFolder;
    for (let n = 2; items.some((i) => i.folder === name); n++) name = `${h.newFolder} ${n}`;
    return name;
  }

  // take removes a site from wherever it is; empty folders go away.
  function take(dir: string) {
    for (let i = items.length - 1; i >= 0; i--) {
      const it = items[i];
      if (it.site === dir) items.splice(i, 1);
      else if (it.folder && it.sites?.includes(dir)) {
        it.sites = it.sites.filter((d) => d !== dir);
        if (!it.sites.length) {
          if (open === it.folder) open = null;
          items.splice(i, 1);
        }
      }
    }
  }

  // --- moving -------------------------------------------------------------
  type Zone = "before" | "after" | "into";
  let dragged = $state("");
  let over = $state<{ key: string; zone: Zone } | null>(null);
  const keyOf = (it: HomeItem) => (it.folder ? "f:" + it.folder : "s:" + it.site);

  function zone(e: DragEvent, canInto: boolean): Zone {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const x = (e.clientX - r.left) / r.width;
    if (!canInto) return x < 0.5 ? "before" : "after";
    return x < 0.22 ? "before" : x > 0.78 ? "after" : "into";
  }

  function dropOn(target: HomeItem, z: Zone) {
    const dir = dragged;
    dragged = "";
    over = null;
    if (!dir || target.site === dir) return;
    const targetKey = keyOf(target);
    take(dir);
    let i = items.findIndex((it) => keyOf(it) === targetKey);
    if (i < 0) i = items.length;
    if (z === "into" && items[i]?.folder) {
      items[i].sites = [...(items[i].sites ?? []), dir];
    } else if (z === "into" && items[i]?.site) {
      const name = newName();
      items[i] = { folder: name, sites: [items[i].site!, dir] };
      renaming = name;
    } else {
      items.splice(z === "after" ? i + 1 : i, 0, { site: dir });
    }
    save();
  }

  // Inside an open folder: only the order changes.
  function dropInFolder(target: string, z: Zone) {
    const dir = dragged;
    dragged = "";
    over = null;
    const f = items[folderAt(open!)];
    if (!f?.sites || !dir || dir === target || !f.sites.includes(dir)) return;
    const rest = f.sites.filter((d) => d !== dir);
    const i = rest.indexOf(target);
    rest.splice(z === "after" ? i + 1 : i, 0, dir);
    f.sites = rest;
    save();
  }

  // --- the "⋯" menu ---------------------------------------------------------
  function moveTo(dir: string, folder: string) {
    take(dir);
    const i = folderAt(folder);
    if (i >= 0) items[i].sites = [...(items[i].sites ?? []), dir];
    menu = "";
    save();
  }
  function newFolderWith(dir: string) {
    const i = items.findIndex((it) => it.site === dir);
    const name = newName();
    if (i >= 0) items[i] = { folder: name, sites: [dir] };
    menu = "";
    renaming = name;
    save();
  }
  function takeOut(dir: string) {
    const from = open;
    take(dir);
    const i = from ? folderAt(from) : -1;
    items.splice(i >= 0 ? i + 1 : items.length, 0, { site: dir });
    menu = "";
    save();
  }
  function rename(oldName: string, value: string) {
    const name = value.trim();
    renaming = "";
    if (!name || name === oldName || items.some((i) => i.folder === name)) return;
    const i = folderAt(oldName);
    if (i < 0) return;
    items[i].folder = name;
    if (open === oldName) open = name;
    save();
  }
  function ungroup(name: string) {
    const i = folderAt(name);
    if (i < 0) return;
    const inside = (items[i].sites ?? []).map((site) => ({ site }));
    items.splice(i, 1, ...inside);
    open = null;
    save();
  }

  let home = $state("");
  HomeDir().then((h) => (home = h));
  const short = (v: string) => (home && v.startsWith(home) ? "~" + v.slice(home.length) : v);
  const blupOf = (s?: SiteInfo) => (s?.model === "bento" ? "turma-4" : "turma-3");
  const folders = $derived(items.filter((i) => i.folder));
  const focus = (el: HTMLInputElement) => { el.focus(); el.select(); };
</script>

<svelte:window onclick={() => (menu = "")} />

{#snippet siteCard(dir: string, inFolder: boolean)}
  {@const s = sites[dir]}
  {#if s}
    <div class="site" class:dragging={dragged === dir} draggable="true"
      ondragstart={(e) => { dragged = dir; e.dataTransfer?.setData("text/plain", dir); }}
      ondragend={() => { dragged = ""; over = null; }}
      ondragover={(e) => { if (!dragged || dragged === dir) return; e.preventDefault(); over = { key: "s:" + dir, zone: zone(e, !inFolder) }; }}
      ondragleave={() => (over = null)}
      ondrop={(e) => { e.preventDefault(); const z = over?.zone ?? "after"; inFolder ? dropInFolder(dir, z) : dropOn({ site: dir }, z); }}
      class:before={over?.key === "s:" + dir && over.zone === "before"}
      class:after={over?.key === "s:" + dir && over.zone === "after"}
      class:into={over?.key === "s:" + dir && over.zone === "into"}>
      <button class="open" onclick={() => onOpen(s)}>
        <img src="/blups/{blupOf(s)}.svg" alt="" />
        <span class="kind">{s.model === "bento" ? t.product : t.blog}</span>
        <strong>{s.title}</strong>
        <span class="path muted" title={s.folder}>{short(s.folder)}</span>
      </button>
      <button class="more" title={h.options} onclick={(e) => { e.stopPropagation(); menu = menu === dir ? "" : dir; }}><Ellipsis size={18} /></button>
      {#if menu === dir}
        <div class="menu" role="menu">
          {#if inFolder}
            <button onclick={() => takeOut(dir)}>{h.takeOut}</button>
          {:else}
            {#each folders as f (f.folder)}<button onclick={() => moveTo(dir, f.folder!)}>{h.moveTo.replace("{name}", f.folder!)}</button>{/each}
            <button onclick={() => newFolderWith(dir)}>{h.newFolderWith}</button>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
{/snippet}

<main class="home">
  <header>
    <img src="/blup.svg" alt="" width="40" height="40" />
    <h1>Blup Studio</h1>
  </header>

  {#if open !== null}
    <section>
      <div class="folder-head">
        <button class="btn ghost" onclick={() => (open = null)}><ArrowLeft size={16} />{h.all}</button>
        {#if renaming === open}
          <input class="name" type="text" value={open} use:focus onblur={(e) => rename(open!, e.currentTarget.value)} onkeydown={(e) => e.key === "Enter" && e.currentTarget.blur()} />
        {:else}
          <button class="name" title={h.rename} onclick={() => (renaming = open!)}>{open}</button>
        {/if}
        <span class="spacer"></span>
        <button class="btn ghost" onclick={() => ungroup(open!)}>{h.ungroup}</button>
      </div>
      <p class="muted hint">{h.folderHint}</p>
      <div class="grid">
        {#each openItems as dir (dir)}{@render siteCard(dir, true)}{/each}
      </div>
    </section>
  {:else}
    <section>
      <h2>{t.sites}</h2>
      <p class="muted">{t.sitesHelp} {items.length > 1 ? h.hint : ""}</p>
      {#if error}<p class="error">{t.error}: {error}</p>{/if}
      <div class="grid">
        <button class="site new" onclick={onNew}>
          <img src="/blups/blup.svg" alt="" />
          <strong>{t.ob.newSite}</strong>
          <span class="muted">{t.ob.newSiteHelp}</span>
        </button>
        {#each items as it (keyOf(it))}
          {#if it.folder}
            {@const f = it.folder}
            <div class="site folder"
              ondragover={(e) => { if (!dragged) return; e.preventDefault(); over = { key: "f:" + f, zone: zone(e, true) }; }}
              ondragleave={() => (over = null)}
              ondrop={(e) => { e.preventDefault(); dropOn(it, over?.zone ?? "into"); }}
              class:before={over?.key === "f:" + f && over.zone === "before"}
              class:after={over?.key === "f:" + f && over.zone === "after"}
              class:into={over?.key === "f:" + f && over.zone === "into"}>
              <button class="open" onclick={() => (open = f)}>
                <span class="minis">{#each (it.sites ?? []).slice(0, 4) as d (d)}<img src="/blups/{blupOf(sites[d])}.svg" alt="" />{/each}</span>
                {#if renaming === f}
                  <input class="name" type="text" value={f} use:focus onclick={(e) => e.stopPropagation()} onblur={(e) => rename(f, e.currentTarget.value)} onkeydown={(e) => e.key === "Enter" && e.currentTarget.blur()} />
                {:else}
                  <strong>{f}</strong>
                {/if}
                <span class="muted">{h.count.replace("{n}", String(it.sites?.length ?? 0))}</span>
              </button>
            </div>
          {:else}
            {@render siteCard(it.site!, false)}
          {/if}
        {/each}
      </div>
    </section>
  {/if}
</main>

<style>
  .home { height: 100%; overflow: auto; padding: 3rem clamp(1.5rem, 6vw, 5rem); }
  header { display: flex; align-items: center; gap: .75rem; margin-block-end: 3rem; }
  h1 { font-size: 1.25rem; font-weight: 700; }
  h2 { font-size: 1.75rem; font-weight: 700; letter-spacing: -.02em; }
  section > p { margin-block: .25rem 1.5rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(17rem, 1fr)); gap: 1rem; }
  .site { position: relative; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); transition: border-color .15s, transform .15s, box-shadow .15s, opacity .15s; }
  .site:hover { border-color: var(--muted); }
  .site.new { display: grid; gap: .35rem; justify-items: start; text-align: start; padding: 1.25rem; border-style: dashed; background: transparent; }
  .site img, .site.new img { width: 2.5rem; }
  .open { width: 100%; display: grid; gap: .35rem; justify-items: start; text-align: start; padding: 1.25rem; padding-inline-end: 2.75rem; }
  .open strong { font-size: 1.05rem; }
  .kind { font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; color: var(--blue); }
  .path { max-width: 100%; font-size: .8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .more { position: absolute; inset-block-start: .75rem; inset-inline-end: .6rem; display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .45rem; color: var(--muted); }
  .more:hover { background: var(--soft); color: var(--text); }
  .menu { position: absolute; z-index: 5; inset-block-start: 2.9rem; inset-inline-end: .6rem; display: grid; min-width: 13rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .12); }
  .menu button { text-align: start; padding: .5rem .7rem; border-radius: .4rem; font-size: .875rem; }
  .menu button:hover { background: var(--soft); }
  .dragging { opacity: .4; }
  .before { box-shadow: -4px 0 0 0 var(--blue); }
  .after { box-shadow: 4px 0 0 0 var(--blue); }
  .into { border-color: var(--blue); box-shadow: 0 0 0 3px color-mix(in srgb, var(--blue) 35%, transparent); transform: scale(1.02); }
  .folder { background: var(--soft); }
  .minis { display: grid; grid-template-columns: repeat(4, 1.6rem); gap: .3rem; }
  .minis img { width: 1.6rem; }
  .folder-head { display: flex; align-items: center; gap: .75rem; margin-block-end: .5rem; }
  .name { font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; padding: .1rem .3rem; border-radius: .4rem; }
  button.name:hover { background: var(--soft); }
  input.name { width: auto; min-width: 12rem; }
  .folder input.name { font-size: 1rem; }
  .spacer { flex: 1; }
  .hint { font-size: .85rem; }
  .error { color: var(--red); }
</style>
