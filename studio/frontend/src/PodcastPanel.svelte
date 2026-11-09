<script lang="ts">
  import { onDestroy } from "svelte";
  import { Podcast, SavePodcast, PickCover, LinkedImage, OpenURL, NarrationPlan } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { PodcastInfo, NarrationPlan as Plan } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { narrateJob, startNarrateSite, cancelNarrateSite, narratePercent, onNarrateEnd } from "./narratejob.svelte";
  import { Clipboard } from "@wailsio/runtime";
  import TextField from "./ui/TextField.svelte";
  import Switch from "./Switch.svelte";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { toast } from "./toast.svelte";
  import { t } from "./i18n";

  // A blog's podcast: the show's details (properties of the Podcast
  // section's note, read by the theme's feed), its cover, how many
  // episodes there are (posts with a narration) and the feed to give to
  // Spotify and Apple Podcasts once.
  const p = t.pod;
  const CATEGORIES = Object.keys(p.cats);
  const NAMES: Record<string, string> = { pt: "Português", en: "English", es: "Español" };

  let info = $state<PodcastInfo | null>(null);
  let error = $state("");
  let title = $state("");
  let description = $state("");
  let author = $state("");
  let email = $state("");
  let category = $state("Technology");
  let explicit = $state(false);
  let cover = $state("");
  const s = saver();
  const img = saver();

  function fill(i: PodcastInfo) {
    info = i;
    title = i.title;
    description = i.description;
    author = i.author;
    email = i.email;
    category = i.category || "Technology";
    explicit = i.explicit;
    preview(i.image);
  }
  async function preview(link: string) {
    cover = link ? await LinkedImage(link).catch(() => "") : "";
  }
  Podcast().then(fill).catch((e) => (error = String(e).replace(/^(Runtime)?Error: /, "")));

  const save = () => s.run(async () => {
    fill(await SavePodcast([
      { key: "podcast_title", value: title },
      { key: "description", value: description },
      { key: "podcast_author", value: author },
      { key: "podcast_email", value: email },
      { key: "podcast_category", value: category },
      { key: "podcast_explicit", value: explicit },
    ]));
  }, p.saved);
  let picked = false;
  const pick = () => img.run(async () => {
    picked = false;
    const link = await PickCover();
    if (!link) return;
    picked = true;
    fill(await SavePodcast([{ key: "image", value: link }]));
  }, () => (picked ? t.toast.imageSet : false));
  const drop = () => img.run(async () => { fill(await SavePodcast([{ key: "image", value: "" }])); }, t.toast.imageRemoved);

  // Apple wants a square of 1400 to 3000 px.
  const coverOff = $derived(!!info && info.imageW > 0 && (info.imageW !== info.imageH || info.imageW < 1400 || info.imageW > 3000));
  const episodes = $derived(!info ? "" : info.episodes === 0 ? p.episodes0 : info.episodes === 1 ? p.episodes1 : p.episodesN.replace("{n}", String(info.episodes)));

  // Narrating every post at once (narratejob): what it would do, read
  // again when a batch ends, and a confirmation before it starts.
  const a = t.narrAll;
  let plan = $state<Plan | null>(null);
  let planError = $state("");
  let confirming = $state(false);
  function loadPlan() {
    NarrationPlan().then((x) => { plan = x; planError = ""; }).catch((e) => (planError = String(e).replace(/^(Runtime)?Error: /, "")));
  }
  loadPlan();
  // A batch that ends: the counts and the episodes are read again.
  onDestroy(onNarrateEnd(() => { loadPlan(); Podcast().then(fill).catch(() => {}); }));
  // About 900 characters a minute, read aloud.
  const minutes = $derived(plan ? Math.max(1, Math.round(plan.chars / 900)) : 0);
  const blocked = $derived(!!plan && (!!plan.costError || (!!plan.cost && !plan.cost.enough)));
  const noteName = (n: string) => n.split("/").pop()?.replace(/\.md$/i, "") ?? "";
  function narrateAll() {
    confirming = false;
    startNarrateSite();
  }

  async function copy(url: string) {
    await Clipboard.SetText(url);
    toast(p.copied);
  }
</script>

<div class="panel">
  <h1>{p.title}</h1>
  <p class="lead">{p.lead}</p>
  {#if error}<p class="bad">{error}</p>{/if}
  {#if info}
    <section class="card">
      <h2>{p.episodes}</h2>
      <p>{episodes}</p>
    </section>

    <section class="card">
      <h2>{a.title}</h2>
      <p class="muted">{a.lead}</p>
      {#if narrateJob.running}
        <div class="nbar" role="progressbar" aria-label={a.title} aria-valuemin="0" aria-valuemax="100" aria-valuenow={narratePercent()}>
          <span style:width="{Math.max(3, narratePercent())}%"></span>
        </div>
        <small class="muted">{narrateJob.total ? a.running.replace("{done}", String(Math.min(narrateJob.done + 1, narrateJob.total))).replace("{total}", String(narrateJob.total)).replace("{note}", noteName(narrateJob.note)) : a.starting}</small>
        <div class="row"><button class="btn ghost" onclick={cancelNarrateSite}>{a.stop}</button></div>
      {:else if planError}
        <p class="bad">{planError}</p>
      {:else if !plan}
        <small class="muted">{a.loading}</small>
      {:else}
        <ul class="narr-sum">
          {#if plan.missing}<li>{a.missing.replace("{n}", String(plan.missing))}</li>{/if}
          {#if plan.outdated}<li>{a.outdated.replace("{n}", String(plan.outdated))}</li>{/if}
          {#each Object.entries(plan.translations ?? {}) as [lang, n] (lang)}<li>{a.translations.replace("{n}", String(n)).replace("{lang}", NAMES[lang] ?? lang)}</li>{/each}
          {#if plan.locked}<li class="muted">{a.locked.replace("{n}", String(plan.locked))}</li>{/if}
        </ul>
        {#if plan.total === 0}
          <p>{a.none}</p>
        {:else}
          {#each plan.voices ?? [] as v (v.lang)}
            <small class="muted">{a.voice.replace("{lang}", NAMES[v.lang] ?? v.lang).replace("{name}", v.name).replace("{engine}", a[v.engine as "local" | "blup" | "openrouter"] ?? v.engine)}</small>
          {/each}
          <small class="muted">{a.minutes.replace("{min}", String(minutes))}</small>
          {#if plan.cost}<small class:bad={!plan.cost.enough} class:muted={plan.cost.enough}>{(plan.cost.enough ? a.cost : a.costLow).replace("{pct}", String(plan.cost.percent)).replace("{left}", String(plan.cost.left))}</small>{/if}
          {#if plan.costError}<small class="bad">{plan.costError}</small>{/if}
          {#if confirming}
            <p>{a.confirm.replace("{n}", String(plan.total))}</p>
            <div class="row">
              <button class="btn" onclick={narrateAll}>{a.yes}</button>
              <button class="btn ghost" onclick={() => (confirming = false)}>{a.no}</button>
            </div>
          {:else}
            <div class="row"><button class="btn" disabled={blocked} onclick={() => (confirming = true)}>{a.button.replace("{n}", String(plan.total))}</button></div>
          {/if}
        {/if}
      {/if}
    </section>

    <section class="card">
      <TextField label={p.name} bind:value={title} placeholder={info.siteName} help={p.nameHelp} />
      <label class="field"><span>{p.desc}</span><textarea bind:value={description}></textarea><small>{p.descHelp}</small></label>
      <TextField label={p.author} bind:value={author} help={p.authorHelp} />
      <TextField label={p.email} type="email" bind:value={email} help={p.emailHelp} />
      <label class="field"><span>{p.category}</span>
        <select bind:value={category}>
          {#each CATEGORIES as c (c)}<option value={c}>{p.cats[c]}</option>{/each}
        </select>
        <small>{p.categoryHelp}</small>
      </label>
      <Switch label={p.explicit} help={p.explicitHelp} bind:checked={explicit} />
      <div class="actions"><Saved state={s.state} /><button class="btn" disabled={s.state.busy} onclick={save}>{t.save}</button></div>
    </section>

    <section class="card">
      <div class="pod-cover">
        <div class="pod-thumb">{#if cover}<img src={cover} alt="" />{:else}<span class="muted">{t.look.none}</span>{/if}</div>
        <div class="field"><span>{p.cover}</span><small>{p.coverHelp}</small>{#if coverOff}<small class="bad">{p.coverWarn.replace("{w}", String(info.imageW)).replace("{h}", String(info.imageH))}</small>{/if}</div>
        <div class="row">
          <button class="btn ghost" disabled={img.state.busy} onclick={pick}>{cover ? t.change : t.choose}</button>
          {#if info.image}<button class="btn ghost" disabled={img.state.busy} onclick={drop}>{t.remove}</button>{/if}
        </div>
      </div>
      <div class="actions"><Saved state={img.state} /></div>
    </section>

    <section class="card">
      <h2>{p.feed}</h2>
      {#if info.hasURL}
        <p>{p.feedHelp}</p>
        {#each info.feeds ?? [] as f (f.lang)}
          <div class="pod-feed">
            {#if (info.feeds ?? []).length > 1}<span class="pod-lang">{NAMES[f.lang] ?? f.lang}</span>{/if}
            <input type="text" readonly value={f.url} aria-label={p.feed} />
            <button class="btn ghost" onclick={() => copy(f.url)}>{p.copy}</button>
          </div>
        {/each}
        <div class="row">
          <button class="btn ghost" onclick={() => OpenURL("https://creators.spotify.com/")}>{p.spotify}</button>
          <button class="btn ghost" onclick={() => OpenURL("https://podcastsconnect.apple.com/")}>{p.apple}</button>
        </div>
        <small class="muted">{p.steps}</small>
      {:else}
        <p>{p.noURL}</p>
      {/if}
    </section>
  {/if}
</div>

<style>
  .pod-cover { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 1rem; }
  .pod-thumb { display: grid; place-items: center; width: 4.5rem; height: 4.5rem; overflow: hidden; border: 1px dashed var(--line); border-radius: .5rem; font-size: .7rem; }
  .pod-thumb img { width: 100%; height: 100%; object-fit: cover; }
  .pod-feed { display: flex; align-items: center; gap: .5rem; }
  .pod-feed input { flex: 1; min-width: 0; }
  .nbar { height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .nbar span { display: block; height: 100%; border-radius: inherit; background: var(--blue); transition: width .3s ease; }
  .narr-sum { list-style: none; padding: 0; margin: 0; display: grid; gap: .2rem; }
  .pod-lang { flex: none; width: 5.5rem; font-size: .85rem; font-weight: 600; }
</style>
