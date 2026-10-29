<script lang="ts">
  import { OpenExisting, HomeDir, Vaults, VaultFolders, SuggestPlaces, CheckSiteDir, NeedsLinkAdjust, PickFolder, CreateSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteInfo, Vault } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Switch from "./Switch.svelte";
  import Check from "@lucide/svelte/icons/check";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";

  // The new site screens: one question per step, like "blup new".
  let { onDone, onCancel, canCancel }: { onDone: (s: SiteInfo) => void; onCancel: () => void; canCancel: boolean } = $props();
  const o = t.ob;

  let step = $state("hello");
  let model = $state<"" | "mochi" | "bento">("");
  let sections = $state<string[]>(["docs", "roadmap", "changelog", "help", "blog"]);
  let title = $state("");
  let description = $state("");
  let author = $state("");
  let lang = $state(t.lang);
  let where = $state<"new" | "vault" | "pick">("new");
  let newFolder = $state("");
  let pickedFolder = $state("");
  let vaults = $state<Vault[]>([]);
  let vault = $state("");
  let folders = $state<string[]>([]);
  let folder = $state("\u0000new"); // a folder of the vault, "" for all of it, or a new one
  let dir = $state("");
  let dirProblem = $state("");
  let showAdvanced = $state(false);
  let adjust = $state(true);
  let needsAdjust = $state(false);
  let error = $state("");
  let created = $state<SiteInfo | null>(null);

  Vaults().then((v) => { vaults = v ?? []; if (vaults.length) { vault = vaults[0].path; loadFolders(); } });
  async function loadFolders() { folders = (await VaultFolders(vault)) ?? []; }

  const steps = $derived(["hello", "theme", ...(model === "bento" ? ["sections"] : []), "name", "notes", "review"]);
  const index = $derived(steps.indexOf(step));
  const total = $derived(steps.length - 1);

  // Suggested places follow the name until the user changes them.
  let placesTouched = false;
  async function suggest() {
    if (placesTouched) return;
    const p = await SuggestPlaces(title.trim() || "site");
    newFolder = p.notes;
    dir = p.dir;
  }

  // The notes: the vault (or folder) and the site's folder inside it.
  const target = $derived.by(() => {
    if (where === "new") return { vault: newFolder, folder: "" };
    if (where === "pick") return { vault: pickedFolder, folder: "" };
    const f = folder === "\u0000new" ? title.trim() || "Site" : folder;
    return { vault, folder: f };
  });

  $effect(() => {
    const v = where === "vault" ? vault : where === "pick" ? pickedFolder : "";
    if (!v) { needsAdjust = false; return; }
    NeedsLinkAdjust(v).then((n) => (needsAdjust = n));
  });

  const ready = $derived.by(() => {
    if (step === "theme") return !!model;
    if (step === "sections") return sections.length > 0;
    if (step === "name") return !!title.trim();
    if (step === "notes") return (where === "new" ? !!newFolder : where === "pick" ? !!pickedFolder : !!vault) && !dirProblem;
    return true;
  });

  // Already have a site (made with the CLI or on another computer)?
  let openMsg = $state("");
  async function openExisting() {
    openMsg = "";
    try {
      const f = await OpenExisting();
      if (f?.site) onDone(f.site);
      else if (f?.folder) { where = "pick"; pickedFolder = f.folder; openMsg = o.noSiteHere; }
    } catch (e) { openMsg = String(e).replace(/^(Runtime)?Error: /, ""); }
  }

  async function next() {
    error = "";
    if (step === "name") await suggest();
    if (step === "notes") {
      dirProblem = await CheckSiteDir(dir);
      if (dirProblem) { showAdvanced = true; return; }
    }
    step = steps[index + 1];
  }
  function back() { error = ""; step = index > 0 ? steps[index - 1] : step; }

  async function pick(kind: "notes" | "dir") {
    const f = await PickFolder().catch(() => "");
    if (!f) return;
    placesTouched = true;
    if (kind === "dir") { dir = f; dirProblem = await CheckSiteDir(dir); }
    else if (where === "new") newFolder = f;
    else pickedFolder = f;
  }

  async function create() {
    step = "creating";
    error = "";
    try {
      created = await CreateSite({
        vault: target.vault, folder: target.folder, dir, model: model || "mochi",
        sections: model === "bento" ? sections : [], title, description, author, lang, adjustLinks: adjust && needsAdjust,
      });
      step = "done";
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
      step = "review";
    }
  }

  const sectionList = $derived([
    ["docs", t.sec.docs, t.sec.docsHelp], ["roadmap", t.sec.roadmap, t.sec.roadmapHelp], ["changelog", t.sec.changelog, t.sec.changelogHelp],
    ["help", t.sec.help, t.sec.helpHelp], ["blog", t.sec.blog, t.sec.blogHelp],
  ]);
  function toggle(k: string, on: boolean) { sections = on ? [...sections, k] : sections.filter((s) => s !== k); }
  let home = $state("");
  HomeDir().then((h) => (home = h));
  // Paths are shown short: "~/Documents/Blup/My site".
  const short = (v: string) => (!v ? "…" : home && v.startsWith(home) ? "~" + v.slice(home.length) : v);
</script>

<div class="ob">
  {#if canCancel && step !== "creating" && step !== "done"}<button class="close btn ghost" onclick={onCancel}>{o.cancel}</button>{/if}

  {#if step === "hello"}
    <div class="hello">
      <img class="bounce big" src="/blups/blup.svg" alt="" />
      <h1>{o.hello}</h1>
      <p class="lead">{o.helloLead}</p>
      <button class="btn big" onclick={next}>{o.start}</button>
      <button class="link-btn" onclick={openExisting}>{o.haveSite}</button>
      {#if openMsg}<p class="muted small">{openMsg}</p>{/if}
    </div>
  {:else if step === "done"}
    <div class="hello">
      <div class="crowd">
        {#each ["turma-3", "blup", "turma-4", "turma-1", "turma-6"] as b, i (b)}<img class="bounce" style:animation-delay="{i * 0.12}s" src="/blups/{b}.svg" alt="" />{/each}
      </div>
      <h1>{o.doneQ}</h1>
      <p class="lead">{o.doneLead}</p>
      <button class="btn big" onclick={() => created && onDone(created)}>{o.open}</button>
    </div>
  {:else if step === "creating"}
    <div class="hello">
      <img class="bounce big" src="/blups/blup.svg" alt="" />
      <h1 class="row"><span class="spin"><Loader size={22} /></span>{o.creating}</h1>
    </div>
  {:else}
    <div class="card-wrap">
      <p class="stepno muted">{o.step.replace("{n}", String(index)).replace("{total}", String(total))}</p>
      <div class="bar"><span style:width="{(index / total) * 100}%"></span></div>

      {#if step === "theme"}
        <h1>{o.themeQ}</h1>
        <div class="themes">
          {#each [["mochi", "turma-3", o.mochi, o.mochiFor, o.mochiAbout, o.mochiHas], ["bento", "turma-4", o.bento, o.bentoFor, o.bentoAbout, o.bentoHas]] as [key, blup, name, kind, about, has] (key)}
            <button class="theme" class:on={model === key} onclick={() => (model = key as "mochi" | "bento")}>
              <div class="shot"><img src="/themes/{key}.webp" alt="" /><img class="mascot" class:bounce={model === key} src="/blups/{blup}.svg" alt="" /></div>
              <div class="info">
                <span class="kind">{kind}</span>
                <b>{name}</b>
                <p>{about}</p>
                <ul>{#each has as h (h)}<li><Check size={14} />{h}</li>{/each}</ul>
              </div>
              {#if model === key}<span class="picked"><Check size={16} /></span>{/if}
            </button>
          {/each}
        </div>
      {:else if step === "sections"}
        <h1>{o.sectionsQ}</h1>
        <p class="lead">{o.sectionsLead}</p>
        <div class="card list">
          {#each sectionList as [k, name, help] (k)}
            <Switch label={name} {help} checked={sections.includes(k)} onchange={(v) => toggle(k, v)} />
          {/each}
          {#if !sections.length}<p class="bad">{o.sectionsNone}</p>{/if}
        </div>
      {:else if step === "name"}
        <h1>{o.nameQ}</h1>
        <p class="lead">{o.nameLead}</p>
        <div class="card">
          <label class="field"><span>{o.name}</span><input class="title" type="text" bind:value={title} placeholder={o.namePh} /></label>
          <label class="field"><span>{o.description}</span><input type="text" bind:value={description} placeholder={o.descriptionPh} /></label>
          <label class="field"><span>{o.author}</span><input type="text" bind:value={author} /><small>{o.authorHelp}</small></label>
          <div class="field"><span>{o.siteLang}</span>
            <div class="langs">
              <button class="choice" class:on={lang === "pt"} onclick={() => (lang = "pt")}>Português</button>
              <button class="choice" class:on={lang === "en"} onclick={() => (lang = "en")}>English</button>
            </div>
          </div>
        </div>
      {:else if step === "notes"}
        <h1>{o.notesQ}</h1>
        <p class="lead">{o.notesLead}</p>
        <div class="wheres">
          <button class="choice where" class:on={where === "new"} onclick={() => (where = "new")}><b>{o.whereNew}</b><span>{o.whereNewHelp}</span></button>
          <button class="choice where" class:on={where === "vault"} onclick={() => (where = "vault")}><b>{o.whereVault}</b><span>{o.whereVaultHelp}</span></button>
          <button class="choice where" class:on={where === "pick"} onclick={() => (where = "pick")}><b>{o.wherePick}</b><span>{o.wherePickHelp}</span></button>
        </div>
        <div class="card">
          {#if where === "new"}
            <div class="row"><code>{short(newFolder)}</code><button class="btn ghost" onclick={() => pick("notes")}>{o.change}</button></div>
          {:else if where === "pick"}
            <div class="row"><code>{short(pickedFolder)}</code><button class="btn ghost" onclick={() => pick("notes")}>{o.pickFolder}</button></div>
          {:else if !vaults.length}
            <p class="muted">{o.noVaults}</p>
          {:else}
            <label class="field"><span>{o.vault}</span>
              <select bind:value={vault} onchange={loadFolders}>{#each vaults as v (v.path)}<option value={v.path}>{v.name}</option>{/each}</select>
            </label>
            <label class="field"><span>{o.folder}</span>
              <select bind:value={folder}>
                <option value={"\u0000new"}>{o.folderNew.replace("{name}", title.trim() || "Site")}</option>
                {#each folders as f (f)}<option value={f}>{f}</option>{/each}
                <option value="">{o.folderAll}</option>
              </select>
            </label>
          {/if}
          {#if needsAdjust}<Switch label={o.adjust} help={o.adjustHelp} bind:checked={adjust} />{/if}
          <button class="add" onclick={() => (showAdvanced = !showAdvanced)}>{showAdvanced ? "−" : "+"} {o.advanced}</button>
          {#if showAdvanced}
            <div class="field"><small>{o.advancedHelp}</small>
              <div class="row"><code title={dir}>{short(dir)}</code><button class="btn ghost" onclick={() => pick("dir")}>{o.change}</button></div>
              {#if dirProblem}<small class="bad">{dirProblem}</small>{/if}
            </div>
          {/if}
        </div>
      {:else if step === "review"}
        <h1>{o.reviewQ}</h1>
        <p class="lead">{o.reviewLead}</p>
        <div class="card review">
          <img class="mascot-r" src="/blups/{model === 'bento' ? 'turma-4' : 'turma-3'}.svg" alt="" />
          <dl>
            <dt>{o.name}</dt><dd>{title}</dd>
            <dt>{o.theme}</dt><dd>{model === "bento" ? `${o.bento} · ${sections.length}` : o.mochi}</dd>
            <dt>{o.notes}</dt><dd><code>{short(target.vault)}{target.folder ? "/" + target.folder : ""}</code></dd>
            <dt>{o.files}</dt><dd><code title={dir}>{short(dir)}</code></dd>
          </dl>
        </div>
        {#if error}<p class="bad">{error}</p>{/if}
      {/if}

      <div class="nav">
        <button class="btn ghost" onclick={back}>{o.back}</button>
        {#if step === "review"}
          <button class="btn big" onclick={create}>{o.create}</button>
        {:else}
          <button class="btn" disabled={!ready} onclick={next}>{o.next}</button>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .ob { position: relative; height: 100%; overflow: auto; display: grid; place-items: center; padding: 3rem 1.5rem; }
  .close { position: absolute; inset-block-start: 1rem; inset-inline-end: 1rem; }
  .hello { display: grid; justify-items: center; gap: 1rem; text-align: center; max-width: 34rem; }
  .hello h1 { font-size: 2rem; font-weight: 800; letter-spacing: -.03em; }
  .lead { color: var(--muted); font-size: 1.05rem; }
  .big { width: 7.5rem; height: auto; }
  .btn.big { width: auto; padding: .75rem 1.6rem; font-size: 1rem; }
  .crowd { display: flex; gap: .5rem; align-items: flex-end; }
  .crowd img { width: 4.5rem; }
  .bounce { animation: bounce 1.6s ease-in-out infinite; }
  @keyframes bounce { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-10px); } }
  .spin { display: inline-grid; animation: spin 1s linear infinite; margin-inline-end: .5rem; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .card-wrap { width: min(100%, 52rem); min-width: 0; display: grid; grid-template-columns: minmax(0, 1fr); gap: 1.25rem; }
  .card-wrap :global(.card), .card-wrap :global(.row) { min-width: 0; }
  .card-wrap > h1 { font-size: 1.75rem; font-weight: 800; letter-spacing: -.02em; }
  .card-wrap > .lead { margin-block-start: -.75rem; }
  .stepno { font-size: .8rem; font-weight: 600; }
  .bar { height: .35rem; border-radius: 999px; background: var(--line); overflow: hidden; margin-block-start: -.75rem; }
  .bar span { display: block; height: 100%; background: var(--blue); transition: width .3s; }
  .themes { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
  .theme { position: relative; display: grid; text-align: start; border: 1px solid var(--line); border-radius: .9rem; background: var(--panel); overflow: hidden; transition: border-color .15s, transform .15s, box-shadow .15s; }
  .theme:hover { transform: translateY(-2px); }
  .theme.on { border-color: var(--blue); box-shadow: 0 0 0 2px var(--blue); }
  .shot { position: relative; aspect-ratio: 16 / 9; background: var(--soft); }
  .shot img:first-child { width: 100%; height: 100%; object-fit: cover; object-position: top; }
  .mascot { position: absolute; inset-block-end: -1.5rem; inset-inline-start: 1rem; width: 4.5rem; }
  .info { display: grid; gap: .4rem; padding: 2rem 1.25rem 1.25rem; }
  .info b { font-size: 1.3rem; }
  .info p { color: var(--muted); font-size: .9rem; }
  .info ul { list-style: none; padding: 0; display: grid; gap: .3rem; font-size: .85rem; }
  .info li { display: flex; gap: .4rem; align-items: center; }
  .info li :global(svg) { color: #16a34a; flex: none; }
  .kind { font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--blue); }
  .picked { position: absolute; inset-block-start: .75rem; inset-inline-end: .75rem; display: grid; place-items: center; width: 1.8rem; height: 1.8rem; border-radius: 50%; background: var(--blue); color: #fff; }
  .list { gap: 1.1rem; }
  input.title { font-size: 1.25rem; font-weight: 600; padding: .7rem .8rem; }
  .langs, .wheres { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; }
  .wheres { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .choice { display: grid; gap: .3rem; text-align: start; padding: .85rem 1rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); font-weight: 600; }
  .choice span { font-size: .8rem; font-weight: 400; color: var(--muted); }
  .choice.on { border-color: var(--blue); box-shadow: inset 0 0 0 1px var(--blue); }
  code { flex: 1; min-width: 0; padding: .5rem .7rem; border-radius: .45rem; background: var(--soft); font-size: .8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .review { grid-template-columns: auto minmax(0, 1fr); align-items: center; gap: 1.5rem; }
  .mascot-r { width: 5rem; }
  dl { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: .5rem 1rem; font-size: .9rem; }
  dt { color: var(--muted); }
  dd { margin: 0; font-weight: 600; min-width: 0; }
  .nav { display: flex; justify-content: space-between; align-items: center; }
  .link-btn { margin-block-start: .75rem; color: var(--blue); font-size: .9rem; font-weight: 600; }
  .link-btn:hover { text-decoration: underline; }
  .small { font-size: .85rem; margin-block-start: .5rem; max-width: 28rem; }
</style>
