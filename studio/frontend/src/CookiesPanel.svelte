<script lang="ts">
  import { SaveCookies, CheckCookieTool } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Switch from "./Switch.svelte";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import X from "@lucide/svelte/icons/x";

  let { settings, reload, onOpenNote }: { settings: Settings; reload: () => void; onOpenNote: (path: string) => void } = $props();

  // The tools: key in blup.toml, field of config.Cookies, label, example.
  const tools = [
    { key: "google_analytics", field: "GoogleAnalytics", label: t.ck.ga, ph: t.ck.gaPh },
    { key: "google_tag_manager", field: "TagManager", label: t.ck.gtm, ph: t.ck.gtmPh },
    { key: "meta_pixel", field: "MetaPixel", label: t.ck.pixel, ph: t.ck.pixelPh },
    { key: "umami", field: "Umami", label: t.ck.umami, ph: t.ck.umamiPh, script: "UmamiScript" },
    { key: "plausible", field: "Plausible", label: t.ck.plausible, ph: t.ck.plausiblePh, script: "PlausibleScript" },
  ] as const;

  const c = $state({ ...settings.cookies }) as Record<string, any>;
  // Tools shown: the ones set, plus the ones being added.
  let shown = $state<string[]>(tools.filter((x) => c[x.field]).map((x) => x.key));
  let errors = $state<Record<string, string>>({});
  let addPick = $state("");
  let page = $state(!settings.cookies.Privacy);
  let contact = $state("");
  const s = saver();

  async function check(key: string, value: string) {
    errors[key] = value.trim() ? (await CheckCookieTool(key, value)) && t.ck.badId : "";
  }
  function remove(x: (typeof tools)[number]) {
    shown = shown.filter((k) => k !== x.key);
    c[x.field] = "";
    if ("script" in x) c[x.script] = "";
    errors[x.key] = "";
  }
  const anyTool = $derived(shown.some((k) => c[tools.find((x) => x.key === k)!.field]?.trim()));

  async function save() {
    await s.run(async () => {
      for (const x of tools) if (errors[x.key]) throw new Error(`${x.label}: ${t.ck.badId}`);
      await SaveCookies(c as any, page, contact);
      reload();
    });
  }
</script>

<div class="panel">
  <h1>{t.ck.title}</h1>
  <p class="lead">{t.ck.lead}</p>

  <section class="card">
    <h2>{t.ck.tools}</h2>
    <p>{t.ck.toolsHelp}</p>
    {#each tools.filter((x) => shown.includes(x.key)) as x (x.key)}
      <div class="tool">
        <label class="field"><span>{x.label}</span>
          <div class="row">
            <input type="text" bind:value={c[x.field]} placeholder={x.ph} oninput={() => check(x.key, c[x.field])} />
            <button class="icon-btn" title={t.remove} onclick={() => remove(x)}><X size={16} /></button>
          </div>
          {#if errors[x.key]}<small class="bad">{errors[x.key]}</small>{/if}
        </label>
        {#if "script" in x}
          <label class="field"><small>{t.ck.script}</small><input type="url" bind:value={c[x.script]} placeholder={t.ck.scriptPh} /></label>
        {/if}
      </div>
    {/each}
    {#if shown.length < tools.length}
      <select bind:value={addPick} onchange={() => { if (addPick) shown = [...shown, addPick]; addPick = ""; }} aria-label={t.ck.addTool}>
        <option value="">+ {t.ck.addTool}</option>
        {#each tools.filter((x) => !shown.includes(x.key)) as x (x.key)}<option value={x.key}>{x.label}</option>{/each}
      </select>
    {/if}
  </section>

  <section class="card">
    <Switch label={t.ck.banner} help={t.ck.bannerHelp} bind:checked={c.Banner} />
    {#if anyTool && !c.Banner}<p class="bad">{t.ck.noBanner}</p>{/if}
  </section>

  {#if c.Banner || anyTool}
    <section class="card">
      <h2>{t.ck.privacy}</h2>
      <label class="radio"><input type="radio" bind:group={page} value={true} /> {t.ck.privacyPage}</label>
      <label class="radio"><input type="radio" bind:group={page} value={false} /> {t.ck.privacyUrl}</label>
      {#if page}
        {#if settings.privacyNote}
          <button class="add" onclick={() => onOpenNote(settings.privacyNote)}>{t.ck.editPolicy} →</button>
        {:else}
          <label class="field"><span>{t.ck.contact}</span><input type="email" bind:value={contact} placeholder="contato@meusite.com.br" /></label>
        {/if}
      {:else}
        <input type="url" bind:value={c.Privacy} placeholder="https://" />
      {/if}
    </section>
  {/if}

  <div class="actions"><Saved state={s.state} /><button class="btn" disabled={s.state.busy} onclick={save}>{t.save}</button></div>
</div>

<style>
  .tool { display: grid; gap: .5rem; padding-block-end: .75rem; border-block-end: 1px dashed var(--line); }
  .radio { display: flex; gap: .5rem; align-items: center; font-size: .9rem; cursor: pointer; }
  select { justify-self: start; width: auto; }
</style>
