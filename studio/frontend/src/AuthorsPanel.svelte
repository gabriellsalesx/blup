<script lang="ts">
  import { Authors, SaveAuthor, DeleteAuthor, PickAuthorPhoto, LinkedImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Author } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import Plus from "@lucide/svelte/icons/plus";
  import X from "@lucide/svelte/icons/x";
  import Pencil from "@lucide/svelte/icons/pencil";

  // The site's authors as cards; one opens as a form (name, photo, role,
  // bio, profiles). Each author is a note in the authors folder, which the
  // site reads; onTree refreshes the files after a change.
  let { onTree }: { onTree?: () => void } = $props();
  const a = t.authors;

  let list = $state<Author[]>([]);
  let photos = $state<Record<string, string>>({});
  let editing = $state<string | null>(null); // the name being edited, "" for a new author
  let form = $state<Author>(blank());
  let social = $state<string[]>([""]); // the form's profile links
  let asking = $state(false);
  const s = saver();

  function blank(): Author { return { name: "", role: "", photo: "", social: [], bio: "", posts: 0, main: false }; }
  async function load() {
    list = (await Authors()) ?? [];
    for (const x of list) if (x.photo && !photos[x.photo]) photos[x.photo] = await LinkedImage(x.photo).catch(() => "");
  }
  load();

  const initials = (n: string) => n.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join("").toUpperCase();
  function edit(x: Author | null) {
    form = x ? { ...x } : blank();
    social = x?.social?.length ? [...x.social] : [""];
    editing = x ? x.name : "";
    asking = false;
  }
  const save = () => s.run(async () => {
    await SaveAuthor(editing ?? "", { ...form, social: social.filter((u) => u.trim()) });
    editing = null;
    await load();
    onTree?.();
  });
  const remove = () => s.run(async () => {
    await DeleteAuthor(editing ?? "");
    editing = null;
    await load();
    onTree?.();
  });
  const pickPhoto = () => s.run(async () => {
    const link = await PickAuthorPhoto(form.name || "author");
    if (!link) return;
    form.photo = link;
    photos[link] = await LinkedImage(link).catch(() => "");
  });
</script>

<div class="panel">
  <h1>{a.title}</h1>
  <p class="lead">{a.lead}</p>

  {#if editing === ""}
    {@render editor()}
  {:else}
    <button class="btn new" onclick={() => edit(null)}><Plus size={16} />{a.add}</button>
  {/if}

  {#each list as x (x.name)}
    {#if editing === x.name}
      {@render editor()}
    {:else}
      <section class="card author">
        <span class="avatar">{#if x.photo && photos[x.photo]}<img src={photos[x.photo]} alt="" />{:else}{initials(x.name)}{/if}</span>
        <div class="who">
          <b>{x.name}{#if x.main}<span class="badge">{a.main}</span>{/if}</b>
          <small class="muted">{[x.role, (x.posts === 1 ? a.post : a.posts.replace("{n}", String(x.posts)))].filter(Boolean).join(" · ")}</small>
        </div>
        <button class="btn ghost" onclick={() => edit(x)}><Pencil size={15} />{a.edit}</button>
      </section>
    {/if}
  {/each}
  {#if !list.length && editing === null}<p class="empty muted">{a.empty}</p>{/if}
</div>

{#snippet editor()}
  <section class="card form">
    <h2>{editing ? a.editTitle.replace("{name}", editing) : a.newTitle}</h2>
    <div class="photo">
      <span class="avatar big">{#if form.photo && photos[form.photo]}<img src={photos[form.photo]} alt="" />{:else}{initials(form.name) || "?"}{/if}</span>
      <div class="row">
        <button class="btn ghost" onclick={pickPhoto}>{form.photo ? t.change : a.choosePhoto}</button>
        {#if form.photo}<button class="btn ghost" onclick={() => (form.photo = "")}>{t.remove}</button>{/if}
      </div>
    </div>
    <label class="field"><span>{a.name}</span><input type="text" bind:value={form.name} placeholder={a.namePh} />{#if editing && form.name.trim() && form.name.trim() !== editing}<small>{a.renameHelp}</small>{/if}</label>
    <label class="field"><span>{a.role}</span><input type="text" bind:value={form.role} placeholder={a.rolePh} /></label>
    <label class="field"><span>{a.bio}</span><textarea rows="4" bind:value={form.bio} placeholder={a.bioPh}></textarea><small>{a.bioHelp}</small></label>
    <div class="field"><span>{a.social}</span>
      {#each social as _, i (i)}
        <div class="row">
          <input type="url" bind:value={social[i]} placeholder="https://instagram.com/…" />
          <button class="icon-btn" title={t.remove} onclick={() => social.splice(i, 1)}><X size={16} /></button>
        </div>
      {/each}
      <button class="add" onclick={() => social.push("")}><Plus size={16} />{a.addSocial}</button>
    </div>
    <div class="actions">
      {#if editing && !asking}<button class="btn ghost danger-text" onclick={() => (asking = true)}>{a.delete}</button>{/if}
      {#if asking}<span class="q">{a.deleteQ}</span><button class="btn ghost" onclick={() => (asking = false)}>{t.sec.cancel}</button><button class="btn danger" onclick={remove}>{a.delete}</button>{/if}
      <span class="grow"></span>
      <Saved state={s.state} />
      <button class="btn ghost" onclick={() => (editing = null)}>{t.sec.cancel}</button>
      <button class="btn" disabled={s.state.busy || !form.name.trim()} onclick={save}>{t.save}</button>
    </div>
  </section>
{/snippet}

<style>
  .new { justify-self: start; }
  .author { grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; }
  .avatar { display: grid; place-items: center; width: 2.75rem; height: 2.75rem; border-radius: 999px; overflow: hidden; background: var(--soft); color: var(--muted); font-weight: 700; font-size: .9rem; flex: none; }
  .avatar.big { width: 4.5rem; height: 4.5rem; font-size: 1.3rem; }
  .avatar img { width: 100%; height: 100%; object-fit: cover; }
  .who { display: grid; gap: .15rem; min-width: 0; }
  .who b { display: flex; align-items: center; gap: .5rem; }
  .badge { padding: .1rem .5rem; border-radius: 999px; background: var(--soft); color: var(--muted); font-size: .7rem; font-weight: 600; }
  .photo { display: flex; align-items: center; gap: 1rem; }
  .actions { display: flex; align-items: center; gap: .5rem; flex-wrap: wrap; }
  .grow { flex: 1; }
  .q { font-size: .85rem; color: var(--muted); }
  .danger-text { color: var(--red); }
  .btn.danger { background: var(--red); color: #fff; }
  .empty { padding: 2rem; border: 1px dashed var(--line); border-radius: .75rem; text-align: center; }
</style>
