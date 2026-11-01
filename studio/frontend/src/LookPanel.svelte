<script lang="ts">
  import { SaveAbout, SaveMenu, SaveSocial, PickImage, RemoveImage, ImageData } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings, MenuItem } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import ArrowUp from "@lucide/svelte/icons/arrow-up";
  import ArrowDown from "@lucide/svelte/icons/arrow-down";
  import X from "@lucide/svelte/icons/x";
  import Plus from "@lucide/svelte/icons/plus";

  let { settings, reload }: { settings: Settings; reload: () => void } = $props();

  // Name and description
  let name = $state(settings.name);
  let description = $state(settings.description);
  let author = $state(settings.author);
  const about = saver();

  // Images
  let images = $state<Record<string, string>>({ logo: "", favicon: "" });
  async function preview(key: "logo" | "favicon", file: string) {
    images[key] = file ? await ImageData(file).catch(() => "") : "";
  }
  preview("logo", settings.logo);
  preview("favicon", settings.favicon);
  const img = saver();
  let picked = false;
  const pick = (key: "logo" | "favicon") => img.run(async () => { picked = false; const f = await PickImage(key); if (f) { picked = true; await preview(key, f); } }, () => (picked ? t.toast.imageSet : false));
  const drop = (key: "logo" | "favicon") => img.run(async () => { await RemoveImage(key); images[key] = ""; }, t.toast.imageRemoved);

  // Menu and button
  let menu = $state<MenuItem[]>((settings.menu ?? []).length ? (settings.menu ?? []).map((m) => ({ ...m })) : (settings.menuOptions ?? []).filter((o) => o.target !== "/tags/").map((o) => ({ ...o })));
  let buttonText = $state(settings.buttonText);
  let buttonLink = $state(settings.buttonLink);
  const menuSave = saver();
  const isLink = (m: MenuItem) => /^(https?:|mailto:)/.test(m.target);
  const unused = $derived((settings.menuOptions ?? []).filter((o) => !menu.some((m) => m.target === o.target)));
  let addPick = $state("");
  function move(i: number, d: number) {
    const j = i + d;
    if (j < 0 || j >= menu.length) return;
    [menu[i], menu[j]] = [menu[j], menu[i]];
  }
  function addPage() {
    const o = (settings.menuOptions ?? []).find((o) => o.target === addPick);
    if (o) menu.push({ ...o });
    addPick = "";
  }

  // Social links
  let social = $state<string[]>((settings.social ?? []).map((s) => s.replace(/^mailto:/, "")));
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
    <div class="actions"><Saved state={img.state} /></div>
  </section>

  <section class="card">
    <h2>{t.look.menu}</h2>
    <p>{t.look.menuHelp}</p>
    {#each menu as m, i (i)}
      <div class="row">
        <input type="text" bind:value={m.name} aria-label={t.look.linkName} />
        {#if isLink(m)}<input type="url" bind:value={m.target} placeholder={t.look.linkUrl} />{:else}<span class="target muted">{m.target}</span>{/if}
        <button class="icon-btn" title={t.up} disabled={i === 0} onclick={() => move(i, -1)}><ArrowUp size={16} /></button>
        <button class="icon-btn" title={t.down} disabled={i === menu.length - 1} onclick={() => move(i, 1)}><ArrowDown size={16} /></button>
        <button class="icon-btn" title={t.remove} onclick={() => menu.splice(i, 1)}><X size={16} /></button>
      </div>
    {/each}
    <div class="row">
      {#if unused.length}
        <select bind:value={addPick} onchange={addPage} aria-label={t.look.addPage}>
          <option value="">+ {t.look.addPage}</option>
          {#each unused as o (o.target)}<option value={o.target}>{o.name}</option>{/each}
        </select>
      {/if}
      <button class="add" onclick={() => menu.push({ name: "", target: "https://" })}><Plus size={16} />{t.look.addLink}</button>
    </div>
    <h2>{t.look.button}</h2>
    <p>{t.look.buttonHelp}</p>
    <div class="row">
      <input type="text" bind:value={buttonText} placeholder={t.look.buttonText} />
      <input type="url" bind:value={buttonLink} placeholder="https://" disabled={!buttonText.trim()} />
    </div>
    <div class="actions"><Saved state={menuSave.state} /><button class="btn" disabled={menuSave.state.busy} onclick={() => menuSave.run(async () => { await SaveMenu(menu, buttonText, buttonLink); reload(); })}>{t.save}</button></div>
  </section>

  <section class="card">
    <h2>{t.look.social}</h2>
    <p>{t.look.socialHelp}</p>
    {#each social as _, i (i)}
      <div class="row">
        <input type="url" bind:value={social[i]} placeholder={t.look.socialPlaceholder} />
        <button class="icon-btn" title={t.remove} onclick={() => social.splice(i, 1)}><X size={16} /></button>
      </div>
    {/each}
    <button class="add" onclick={() => social.push("")}><Plus size={16} />{t.look.addSocial}</button>
    <div class="actions"><Saved state={socialSave.state} /><button class="btn" disabled={socialSave.state.busy} onclick={() => socialSave.run(async () => { await SaveSocial(social); reload(); })}>{t.save}</button></div>
  </section>
</div>

<style>
  .image { display: grid; grid-template-columns: 4.5rem minmax(0, 1fr) auto; gap: 1rem; align-items: center; }
  .thumb { width: 4.5rem; height: 4.5rem; display: grid; place-items: center; border: 1px dashed var(--line); border-radius: .5rem; overflow: hidden; font-size: .7rem; }
  .thumb.round { border-radius: 1rem; }
  .thumb img { max-width: 100%; max-height: 100%; object-fit: contain; }
  .target { flex: 1; min-width: 0; font-size: .85rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding-inline: .5rem; }
</style>
