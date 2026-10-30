<script lang="ts">
  import { SetLanguage, Connections, ConnectGitHub, OpenKeyPage, CloudSites, SetGeneralPassword, AutostartEnabled, SetAutostart, SetTray, SetToggleHotkey } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { CloudSite } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import External from "@lucide/svelte/icons/external-link";
  import Check from "@lucide/svelte/icons/check";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import Switch from "./Switch.svelte";
  import KeyRecorder from "./KeyRecorder.svelte";
  import { t, setLanguage } from "./i18n";
  import { look, setPref } from "./theme.svelte";

  // The first time the Studio opens on a computer: the language (pictures
  // only, the person may read neither), GitHub (and the sites already
  // there), the general password of posts and the window. Each step can be
  // skipped; the choices are saved as they are made.
  let { onDone }: { onDone: () => void } = $props();

  let lang = $state("");
  let step = $state("lang");
  let error = $state("");
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  // GitHub, and the Blup sites found there (only shown here; New site
  // brings them).
  let login = $state("");
  let ghKey = $state("");
  let connecting = $state(false);
  let searching = $state(false);
  let found = $state<CloudSite[] | null>(null);

  const steps = $derived(["lang", "github", ...(found?.length ? ["found"] : []), "password", "window", "done"]);
  const index = $derived(steps.indexOf(step));
  const next = () => { error = ""; step = steps[index + 1]; };
  const back = () => { error = ""; step = steps[Math.max(0, index - 1)]; };

  async function pickLang(l: string) {
    try { await SetLanguage(l); } catch {}
    setLanguage(l);
    lang = l;
    step = "github";
    try {
      const c = await Connections();
      if (c.github) { login = c.github; lookFor(); }
    } catch {}
  }

  async function connect() {
    connecting = true;
    error = "";
    try {
      login = await ConnectGitHub(ghKey.trim());
      ghKey = "";
      lookFor();
    } catch (e) { error = clean(e); } finally { connecting = false; }
  }
  async function lookFor() {
    searching = true;
    try { found = (await CloudSites()) ?? []; } catch { found = []; } finally { searching = false; }
  }

  let pw = $state("");
  let showPw = $state(false);
  async function savePassword() {
    try { await SetGeneralPassword(pw); pw = ""; next(); } catch (e) { error = clean(e); }
  }

  let autostart = $state(false);
  AutostartEnabled().then((v) => (autostart = v)).catch(() => {});
  const desk = async (fn: () => Promise<void>) => { error = ""; try { await fn(); } catch (e) { error = clean(e); } };
  const setAutostart = (v: boolean) => desk(async () => { await SetAutostart(v); autostart = v; });
  const setTray = (v: boolean) => desk(async () => { await SetTray(v); look.tray = v; });
  const setToggle = (keys: string) => desk(async () => { await SetToggleHotkey(keys); look.toggleKey = keys; });

  function finish() {
    setPref("shortcutTip", true);
    setPref("welcomed", true);
    onDone();
  }
</script>

<div class="wel">
  {#if step === "lang"}
    <div class="center">
      <img class="scene" src="/scenes/boas-vindas-idioma.svg" alt="" />
      <div class="langs">
        <button class="lang" onclick={() => pickLang("pt")}><span class="flag br" aria-hidden="true"></span>Português (Brasil)</button>
        <button class="lang" onclick={() => pickLang("en")}><span class="flag us" aria-hidden="true"></span>English</button>
      </div>
    </div>
  {:else}
    <!-- Rebuilt with the language just chosen. -->
    {#key lang}
      <div class="center">
        <!-- The progress, and skipping (discreet, but right where the eyes are). -->
        <div class="top">
          <span></span>
          <div class="dots" aria-hidden="true">{#each steps.slice(1) as s, i (s)}<span class:on={i < index}></span>{/each}</div>
          {#if step === "github" || step === "password"}<button class="skip" onclick={next}>{t.wel.skip}</button>{:else}<span></span>{/if}
        </div>

        {#if step === "github"}
          <img class="scene" src="/scenes/boas-vindas-github.svg" alt="" />
          <h1>{t.wel.githubQ}</h1>
          <p class="lead">{t.wel.githubLead}</p>
          {#if login}
            <p class="ok"><Check size={16} />{t.wel.connectedAs.replace("{user}", login)}</p>
            {#if searching}<p class="muted row"><span class="spin"><Loader size={16} /></span>{t.wel.searching}</p>
            {:else if found && !found.length}<p class="muted">{t.wel.foundNone}</p>{/if}
          {:else}
            <ol class="steps">
              <li>{t.pubs.step1} <button class="add" onclick={() => OpenKeyPage("github")}>{t.pubs.openForm} <External size={14} /></button></li>
              <li>{t.pubs.step2gh}</li>
              <li>{t.pubs.step3}</li>
            </ol>
            <div class="key">
              <input type="password" bind:value={ghKey} placeholder={t.pubs.pasteKey} autocomplete="off" />
              <button class="btn" disabled={connecting || !ghKey.trim()} onclick={connect}>{connecting ? t.pubs.checking : t.pubs.connect}</button>
            </div>
          {/if}
        {:else if step === "found" && found}
          <img class="scene" src="/scenes/boas-vindas-encontrados.svg" alt="" />
          <h1>{found.length === 1 ? t.wel.foundOne : t.wel.foundMany.replace("{n}", String(found.length))}</h1>
          <p class="lead">{t.wel.foundLead}</p>
          <ul class="found">
            {#each found as f (f.owner + "/" + f.repo)}
              <li><img src="/blups/{({ bento: 'turma-4', nori: 'turma-2' } as Record<string, string>)[f.model] ?? 'turma-3'}.svg" alt="" /><b>{f.title}</b><small class="muted">{f.model === "bento" ? t.wel.product : t.wel.blog}</small></li>
            {/each}
          </ul>
        {:else if step === "password"}
          <img class="scene" src="/scenes/boas-vindas-senha.svg" alt="" />
          <h1>{t.wel.pwQ}</h1>
          <p class="lead">{t.wel.pwLead}</p>
          <div class="key">
            <input type={showPw ? "text" : "password"} bind:value={pw} placeholder={t.wel.pwLabel} aria-label={t.wel.pwLabel} autocomplete="new-password" />
            <button class="btn ghost" onclick={() => (showPw = !showPw)}>{showPw ? t.wel.hide : t.wel.show}</button>
          </div>
        {:else if step === "window"}
          <img class="scene" src="/scenes/boas-vindas-janela.svg" alt="" />
          <h1>{t.wel.windowQ}</h1>
          <p class="lead">{t.wel.windowLead}</p>
          <div class="card">
            <Switch label={t.keys.autostart} help={t.keys.autostartHelp} checked={autostart} onchange={setAutostart} />
            <Switch label={t.keys.tray} help={t.keys.trayHelp} checked={!!look.tray} onchange={setTray} />
            <div class="short">
              <div><b>{t.wel.shortcut}</b><small class="muted">{t.wel.shortcutHelp}</small></div>
              <KeyRecorder value={look.toggleKey ?? ""} onChange={setToggle} />
            </div>
          </div>
        {:else if step === "done"}
          <div class="crowd">
            {#each ["turma-3", "blup", "turma-4", "turma-1", "turma-6"] as b, i (b)}<img class="bounce" style:animation-delay="{i * 0.12}s" src="/blups/{b}.svg" alt="" />{/each}
          </div>
          <h1>{t.wel.doneQ}</h1>
          <p class="lead">{t.wel.doneLead}</p>
        {/if}

        {#if error}<p class="bad">{error}</p>{/if}

        <div class="nav">
          <button class="btn ghost" onclick={back}>{t.wel.back}</button>
          {#if step === "github"}
            <button class="btn" disabled={!login || searching} onclick={next}>{t.wel.next}</button>
          {:else if step === "password"}
            <button class="btn" disabled={!pw} onclick={savePassword}>{t.wel.saveNext}</button>
          {:else if step === "done"}
            <button class="btn big" onclick={finish}>{t.wel.start}</button>
          {:else}
            <button class="btn" onclick={next}>{t.wel.next}</button>
          {/if}
        </div>
      </div>
    {/key}
  {/if}
</div>

<style>
  .wel { position: relative; height: 100%; overflow: auto; display: grid; place-items: center; padding: 3rem 1.5rem; }
  .center { width: min(100%, 34rem); display: grid; justify-items: center; gap: 1rem; text-align: center; }
  .scene { width: min(100%, 26rem); aspect-ratio: 320 / 200; }
  h1 { font-size: 1.85rem; font-weight: 800; letter-spacing: -.03em; }
  .lead { color: var(--muted); font-size: 1.02rem; margin-block-start: -.4rem; }
  .langs { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; width: 100%; }
  .lang { display: grid; justify-items: center; gap: .65rem; padding: 1.1rem 1rem; border: 1px solid var(--line); border-radius: .9rem; background: var(--panel); font-weight: 700; transition: border-color .15s, transform .15s; }
  .lang:hover { border-color: var(--blue); transform: translateY(-2px); }
  /* Flags drawn in CSS: no image, no text. */
  .flag { width: 2.6rem; height: 1.8rem; border-radius: .3rem; box-shadow: 0 0 0 1px rgb(0 0 0 / .12); }
  .flag { position: relative; }
  .flag.br { background: #16a34a; }
  .flag.br::before { content: ""; position: absolute; inset: .28rem .35rem; background: #facc15; clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%); }
  .flag.br::after { content: ""; position: absolute; inset: 0; width: .8rem; height: .8rem; margin: auto; border-radius: 50%; background: #1e40af; }
  .flag.us { background: linear-gradient(#1e3a8a, #1e3a8a) 0 0 / 45% 54% no-repeat, repeating-linear-gradient(#dc2626 0 .1385rem, #fff .1385rem .277rem); }
  .dots { display: flex; gap: .4rem; }
  .dots span { width: 1.6rem; height: .3rem; border-radius: 999px; background: var(--line); transition: background .2s; }
  .dots span.on { background: var(--blue); }
  .steps { display: grid; gap: .45rem; padding-inline-start: 1.2rem; text-align: start; font-size: .92rem; width: 100%; }
  .add { display: inline-flex; align-items: center; gap: .3rem; color: var(--blue); font-weight: 600; }
  .key { display: flex; gap: .5rem; width: 100%; }
  .key input { flex: 1; min-width: 0; }
  .ok { display: inline-flex; align-items: center; gap: .4rem; color: #16a34a; font-weight: 600; }
  .row { display: inline-flex; align-items: center; gap: .4rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .found { list-style: none; padding: 0; display: grid; gap: .4rem; width: 100%; }
  .found li { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .7rem; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); text-align: start; }
  .found img { width: 1.9rem; }
  .found b { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .card { display: grid; gap: 1.1rem; width: 100%; padding: 1.1rem 1.2rem; border: 1px solid var(--line); border-radius: .8rem; background: var(--panel); text-align: start; }
  .short { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
  .short div { display: grid; gap: .15rem; }
  .short small { font-size: .82rem; }
  .crowd { display: flex; gap: .5rem; align-items: flex-end; }
  .crowd img { width: 4.5rem; }
  .bounce { animation: bounce 1.6s ease-in-out infinite; }
  @keyframes bounce { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-10px); } }
  .nav { display: flex; justify-content: space-between; align-items: center; width: 100%; margin-block-start: .5rem; }
  .top { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; width: 100%; }
  .skip { justify-self: end; padding: .3rem .6rem; border-radius: .4rem; color: var(--muted); font-size: .9rem; font-weight: 600; }
  .skip:hover { color: var(--text); background: var(--soft); }
  .btn.big { padding: .75rem 1.6rem; font-size: 1rem; }
  .bad { color: var(--red); font-size: .9rem; }
</style>
