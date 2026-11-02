<script lang="ts">
  import { untrack } from "svelte";
  import { SaveAbout, SaveMenu, SaveSocial, PickImage, RemoveImage, ImageData, SetShowName, SaveLogoLink } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import Switch from "./Switch.svelte";
  import SocialLinks from "./ui/SocialLinks.svelte";
  import TextField from "./ui/TextField.svelte";
  import { validLogoLink } from "./logolink";
  import MenuEditor, { toGroups, toMenu, newGroup } from "./ui/MenuEditor.svelte";
  import { NETWORKS } from "./social";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import Plus from "@lucide/svelte/icons/plus";
  import ListPlus from "@lucide/svelte/icons/list-plus";

  let { settings, reload }: { settings: Settings; reload: () => void } = $props();
  // The forms start from the settings as the screen opened; each save
  // reloads them.
  const start = untrack(() => settings);

  // Name and description
  let name = $state(start.name);
  let description = $state(start.description);
  let author = $state(start.author);
  const about = saver();

  // Images
  let images = $state<Record<string, string>>({ logo: "", favicon: "" });
  async function preview(key: "logo" | "favicon", file: string) {
    images[key] = file ? await ImageData(file).catch(() => "") : "";
  }
  preview("logo", start.logo);
  preview("favicon", start.favicon);
  const img = saver();
  let picked = false;
  const pick = (key: "logo" | "favicon") => img.run(async () => { picked = false; const f = await PickImage(key); if (f) { picked = true; await preview(key, f); } }, () => (picked ? t.toast.imageSet : false));
  const drop = (key: "logo" | "favicon") => img.run(async () => { await RemoveImage(key); images[key] = ""; }, t.toast.imageRemoved);

  // Menu and button
  let menu = $state(toGroups((start.menu ?? []).length ? start.menu ?? [] : (start.menuOptions ?? []).filter((o) => o.target !== "/tags/")));
  let buttonText = $state(start.buttonText);
  let buttonLink = $state(start.buttonLink);
  const menuSave = saver();
  const unused = $derived((settings.menuOptions ?? []).filter((o) => !menu.some((g) => g.item.target === o.target || g.kids.some((k) => k.target === o.target))));
  let addPick = $state("");
  function addPage() {
    const o = (settings.menuOptions ?? []).find((o) => o.target === addPick);
    if (o) menu.push(newGroup(o));
    addPick = "";
  }

  // The site's name next to the logo in the header, or the logo alone.
  let showName = $state(start.showName);
  const nameSave = saver();
  const setShowName = (v: boolean) => nameSave.run(async () => { await SetShowName(v); reload(); });

  // The logo's own link (the company's main site, say): off, the logo
  // leads home. Saved only when valid; turning it off saves "".
  const savedLink = () => settings.logoLink ?? "";
  let ownLink = $state(!!savedLink());
  let logoLink = $state(savedLink());
  const linkSave = saver();
  const linkError = $derived(logoLink.trim() && !validLogoLink(logoLink.trim()) ? t.look.logoLinkBad : "");
  function setOwnLink(on: boolean) {
    if (!on && settings.logoLink) linkSave.run(async () => { await SaveLogoLink(""); logoLink = ""; reload(); });
  }
  function saveLogoLink() {
    const v = logoLink.trim();
    if (!v || linkError || v === settings.logoLink) return;
    linkSave.run(async () => { await SaveLogoLink(v); reload(); });
  }

  // Social links
  // As the author's profile: each link with its network's icon (an e-mail
  // keeps its mailto: so it shows the e-mail icon).
  let social = $state<string[]>([...(start.social ?? [])]);
  // A network picked but left with only the start of its address is dropped.
  const filled = () => social.map((u) => u.trim()).filter((u) => u && !NETWORKS.some((n) => n.prefix === u));
  const socialSave = saver();
</script>

<div class="panel">
  <h1>{t.look.title}</h1>
  <p class="lead">{t.look.lead}</p>

  <section class="card">
    <h2>{t.look.about}</h2>
    <label class="field"><span>{t.look.name}</span><input type="text" bind:value={name} /></label>
    <label class="field"><span>{t.look.description}</span><textarea bind:value={description}></textarea><small>{t.look.descriptionHelp}</small></label>
    <label class="field"><span>{t.look.author}</span><input type="text" bind:value={author} /><small>{t.look.authorHelp}</small></label>
    <div class="actions"><Saved state={about.state} /><button class="btn" disabled={about.state.busy} onclick={() => about.run(async () => { await SaveAbout(name, description, author); reload(); })}>{t.save}</button></div>
  </section>

  <section class="card">
    <h2>{t.look.images}</h2>
    {#each [["logo", t.look.logo, t.look.logoHelp], ["favicon", t.look.favicon, t.look.faviconHelp]] as [key, label, help] (key)}
      <div class="image">
        <div class="thumb" class:round={key === "favicon"}>{#if images[key]}<img src={images[key]} alt="" />{:else}<span class="muted">{t.look.none}</span>{/if}</div>
        <div class="field"><span>{label}</span><small>{help}</small></div>
        <div class="row">
          <button class="btn ghost" onclick={() => pick(key as "logo" | "favicon")}>{images[key] ? t.change : t.choose}</button>
          {#if images[key]}<button class="btn ghost" onclick={() => drop(key as "logo" | "favicon")}>{t.remove}</button>{/if}
        </div>
      </div>
    {/each}
    <Switch label={t.look.showName} help={t.look.showNameHelp} bind:checked={showName} onchange={setShowName} />
    <Switch label={t.look.logoLink} help={t.look.logoLinkHelp} bind:checked={ownLink} onchange={setOwnLink} />
    {#if ownLink}
      <TextField label={t.look.logoLinkUrl} bind:value={logoLink} help={t.look.logoLinkUrlHelp} error={linkError} placeholder="https://" spellcheck="false" autocomplete="off" onblur={saveLogoLink} onkeydown={(e) => e.key === "Enter" && saveLogoLink()} />
    {/if}
    <div class="actions"><Saved state={img.state} /><Saved state={nameSave.state} /><Saved state={linkSave.state} /></div>
  </section>

  <section class="card">
    <h2>{t.look.menu}</h2>
    <p>{t.look.menuHelp}</p>
    <MenuEditor bind:groups={menu} />
    <div class="row">
      {#if unused.length}
        <select bind:value={addPick} onchange={addPage} aria-label={t.look.addPage}>
          <option value="">+ {t.look.addPage}</option>
          {#each unused as o (o.target)}<option value={o.target}>{o.name}</option>{/each}
        </select>
      {/if}
      <button class="add" onclick={() => menu.push(newGroup({ name: "", target: "https://" }))}><Plus size={16} />{t.look.addLink}</button>
      <button class="add" onclick={() => menu.push(newGroup({ name: "", target: "" }))}><ListPlus size={16} />{t.look.addGroup}</button>
    </div>
    <h2>{t.look.button}</h2>
    <p>{t.look.buttonHelp}</p>
    <div class="row">
      <input type="text" bind:value={buttonText} placeholder={t.look.buttonText} />
      <input type="url" bind:value={buttonLink} placeholder="https://" disabled={!buttonText.trim()} />
    </div>
    <div class="actions"><Saved state={menuSave.state} /><button class="btn" disabled={menuSave.state.busy} onclick={() => menuSave.run(async () => { await SaveMenu(toMenu(menu), buttonText, buttonLink); reload(); })}>{t.save}</button></div>
  </section>

  <section class="card">
    <h2>{t.look.social}</h2>
    <p>{t.look.socialHelp}</p>
    <SocialLinks bind:links={social} add={t.authors.addSocial} />
    <div class="actions"><Saved state={socialSave.state} /><button class="btn" disabled={socialSave.state.busy} onclick={() => socialSave.run(async () => { await SaveSocial(filled()); reload(); })}>{t.save}</button></div>
  </section>
</div>

<style>
  .image { display: grid; grid-template-columns: 4.5rem minmax(0, 1fr) auto; gap: 1rem; align-items: center; }
  .thumb { width: 4.5rem; height: 4.5rem; display: grid; place-items: center; border: 1px dashed var(--line); border-radius: .5rem; overflow: hidden; font-size: .7rem; }
  .thumb.round { border-radius: 1rem; }
  .thumb img { max-width: 100%; max-height: 100%; object-fit: contain; }
</style>
