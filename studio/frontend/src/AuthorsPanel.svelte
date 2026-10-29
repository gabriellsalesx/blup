<script lang="ts">
  import { NextAuthorAvatar, Authors, SaveAuthor, DeleteAuthor, PickAuthorPhoto, LinkedImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Author } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import Plus from "@lucide/svelte/icons/plus";
  import X from "@lucide/svelte/icons/x";
  import Pencil from "@lucide/svelte/icons/pencil";
  import { tick } from "svelte";
  import { NETWORKS, networkOf } from "./social";

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

  function blank(): Author { return { name: "", role: "", photo: "", social: [], bio: "", posts: 0, main: false, avatar: "" }; }
  // Without a photo, a Blup (it becomes the photo when saved; a new author
  // gets the next one).
  const blup = (file: string) => `/blups/${file || "blup.svg"}`;
  const own = (photo: string) => !!photo && !photo.toLowerCase().endsWith(".svg]]"); // a Blup is an .svg
  async function load() {
    list = (await Authors()) ?? [];
    for (const x of list) if (x.photo && !photos[x.photo]) photos[x.photo] = await LinkedImage(x.photo).catch(() => "");
  }
  load();

  function edit(x: Author | null) {
    form = x ? { ...x } : blank();
    social = x?.social?.length ? [...x.social] : [];
    editing = x ? x.name : "";
    asking = false;
    if (!x) NextAuthorAvatar().then((f) => (form.avatar = f));
  }
  const save = () => s.run(async () => {
    await SaveAuthor(editing ?? "", { ...form, social: social.map((u) => u.trim()).filter((u) => u && !NETWORKS.some((n) => n.prefix === u)) });
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
  // A profile: pick the network, then type the rest of the link.
  async function addNetwork(prefix: string) {
    social.push(prefix);
    await tick();
    const inputs = document.querySelectorAll<HTMLInputElement>(".form .profile input");
    const el = inputs[inputs.length - 1];
    el?.focus();
    el?.setSelectionRange(el.value.length, el.value.length);
  }
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
        <span class="avatar"><img src={x.photo && photos[x.photo] ? photos[x.photo] : blup(x.avatar)} alt="" class:blup={!own(x.photo)} /></span>
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
      <span class="avatar big"><img src={form.photo && photos[form.photo] ? photos[form.photo] : blup(form.avatar)} alt="" class:blup={!own(form.photo)} /></span>
      <div class="row">
        <button class="btn ghost" onclick={pickPhoto}>{own(form.photo) ? t.change : a.choosePhoto}</button>
        {#if own(form.photo)}<button class="btn ghost" onclick={() => (form.photo = "")}>{t.remove}</button>{/if}
      </div>
    </div>
    <label class="field"><span>{a.name}</span><input type="text" bind:value={form.name} placeholder={a.namePh} />{#if editing && form.name.trim() && form.name.trim() !== editing}<small>{a.renameHelp}</small>{/if}</label>
    <label class="field"><span>{a.role}</span><input type="text" bind:value={form.role} placeholder={a.rolePh} /></label>
    <label class="field"><span>{a.bio}</span><textarea rows="4" bind:value={form.bio} placeholder={a.bioPh}></textarea><small>{a.bioHelp}</small></label>
    <div class="field"><span>{a.social}</span>
      {#each social as _, i (i)}
        {@const net = networkOf(social[i])}
        <div class="row profile">
          <span class="net" title={net.name}>{@html net.icon}</span>
          <input type="text" bind:value={social[i]} aria-label={net.name} />
          <button class="icon-btn" title={t.remove} onclick={() => social.splice(i, 1)}><X size={16} /></button>
        </div>
      {/each}
      <small>{a.addSocial}</small>
      <div class="nets">
        {#each NETWORKS as n (n.key)}
          <button class="net-btn" onclick={() => addNetwork(n.prefix)}>{@html n.icon}{n.name}</button>
        {/each}
      </div>
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
  /* The Blup drawings have room around them: a bit bigger fills the circle. */
  .avatar img.blup { object-fit: contain; transform: scale(1.1) translateY(3%); }
  .avatar:has(img.blup) { background: color-mix(in srgb, var(--blue) 12%, transparent); }
  .who { display: grid; gap: .15rem; min-width: 0; }
  .who b { display: flex; align-items: center; gap: .5rem; }
  .badge { padding: .1rem .5rem; border-radius: 999px; background: var(--soft); color: var(--muted); font-size: .7rem; font-weight: 600; }
  .photo { display: flex; align-items: center; gap: 1rem; }
  .actions { display: flex; align-items: center; gap: .5rem; flex-wrap: wrap; }
  .grow { flex: 1; }
  .q { font-size: .85rem; color: var(--muted); }
  .danger-text { color: var(--red); }
  .btn.danger { background: var(--red); color: #fff; }
  .profile .net { display: grid; place-items: center; width: 2.25rem; height: 2.25rem; flex: none; border-radius: .5rem; background: var(--soft); }
  .nets { display: flex; flex-wrap: wrap; gap: .35rem; }
  .net-btn { display: inline-flex; align-items: center; gap: .4rem; padding: .35rem .7rem; border: 1px solid var(--line); border-radius: 999px; font-size: .82rem; font-weight: 600; }
  .net-btn:hover { border-color: var(--blue); color: var(--blue); }
  .empty { padding: 2rem; border: 1px dashed var(--line); border-radius: .75rem; text-align: center; }
</style>
