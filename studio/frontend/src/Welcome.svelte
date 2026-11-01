<script lang="ts">
  import { BlupSites, BringBlupSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupAccount, BlupSite } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import CloudAccount from "./CloudAccount.svelte";
  import { t } from "./i18n";
  import { setPref } from "./theme.svelte";

  // The first time the Studio opens on a computer, as Duolingo does: start
  // right away (the first site), or bring the sites of an account already
  // used elsewhere. The account is optional; it is offered again, gently,
  // once there is a site. The language follows the computer's.
  let { onStart, onOpen }: { onStart: () => void; onOpen: (site: SiteInfo) => void } = $props();
  const h = t.hello;

  let step = $state<"start" | "account" | "bring">("start");
  let account = $state<BlupAccount | null>(null);
  let sites = $state<BlupSite[] | null>(null);
  let bringing = $state("");
  let error = $state("");
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  function done() {
    setPref("welcomed", true);
  }
  function start() {
    done();
    onStart();
  }

  // Signed in with the notes open: the sites kept on Blup Cloud.
  $effect(() => {
    if (step === "account" && account?.notes === "ready") {
      step = "bring";
      BlupSites().then((s) => (sites = s ?? [])).catch((e) => { error = clean(e); sites = []; });
    }
  });

  async function bring(s: BlupSite) {
    bringing = s.vault; error = "";
    try {
      const site = await BringBlupSite(s.vault);
      done();
      onOpen(site);
    } catch (e) { error = clean(e); } finally { bringing = ""; }
  }
</script>

<div class="wel">
  <div class="center">
    {#if step === "start"}
      <img class="scene" src="/scenes/blog-pessoal.svg" alt="" />
      <h1>{h.title}</h1>
      <p class="lead">{h.lead}</p>
      <div class="choices">
        <button class="btn big" onclick={start}>{h.start}</button>
        <button class="btn ghost big" onclick={() => (step = "account")}>{h.account}</button>
      </div>
    {:else if step === "account"}
      <img class="scene" src="/scenes/boas-vindas-encontrados.svg" alt="" />
      <h1>{h.accountTitle}</h1>
      <p class="lead">{h.accountLead} {t.cloud.signInHelp}</p>
      <div class="box" class:plain={!account?.signedIn}><CloudAccount bind:account bare /></div>
      <div class="nav">
        <button class="btn ghost" onclick={() => (step = "start")}>{t.wel.back}</button>
        <button class="link" onclick={start}>{h.instead}</button>
      </div>
    {:else}
      <img class="scene" src="/scenes/boas-vindas-encontrados.svg" alt="" />
      {#if sites === null}
        <p class="muted row"><span class="spin"><Loader size={16} /></span>{h.looking}</p>
      {:else if sites.length}
        <h1>{h.bringTitle}</h1>
        <p class="lead">{h.bringLead}</p>
        <ul class="found">
          {#each sites as s (s.vault)}
            <li>
              <img src="/blups/turma-3.svg" alt="" />
              <b>{s.title}</b>
              {#if s.dir}<small class="muted">{h.here}</small>
              {:else}<button class="btn" disabled={!!bringing} onclick={() => bring(s)}>{#if bringing === s.vault}<span class="spin"><Loader size={14} /></span>{h.bringing}{:else}{h.bring}{/if}</button>{/if}
            </li>
          {/each}
        </ul>
      {:else}
        <h1>{h.noneTitle}</h1>
        <p class="lead">{h.noneLead}</p>
      {/if}
      <div class="nav">
        <button class="btn ghost" onclick={() => (step = "account")}>{t.wel.back}</button>
        <button class="btn" onclick={start}>{h.fresh}</button>
      </div>
    {/if}
    {#if error}<p class="bad">{error}</p>{/if}
    {#if step !== "bring"}<p class="note">{h.optional}</p>{/if}
  </div>
</div>

<style>
  .wel { position: relative; height: 100%; overflow: auto; display: grid; place-items: center; padding: 3rem 1.5rem; }
  .center { width: min(100%, 34rem); display: grid; justify-items: center; gap: 1rem; text-align: center; }
  .scene { width: min(100%, 24rem); aspect-ratio: 320 / 200; }
  h1 { font-size: 1.85rem; font-weight: 800; letter-spacing: -.03em; }
  .lead { color: var(--muted); font-size: 1.02rem; margin-block-start: -.4rem; }
  .choices { display: grid; gap: .6rem; width: min(100%, 22rem); margin-block-start: .5rem; }
  .btn.big { padding: .8rem 1.6rem; font-size: 1rem; justify-content: center; }
  .note { color: var(--muted); font-size: .85rem; }
  .box { width: 100%; display: grid; gap: .75rem; padding: 1.1rem 1.2rem; border: 1px solid var(--line); border-radius: .8rem; background: var(--panel); text-align: start; }
  .box.plain { padding: 0; border: none; background: none; width: min(100%, 22rem); }
  .nav { display: flex; justify-content: space-between; align-items: center; width: 100%; margin-block-start: .5rem; }
  .link { color: var(--blue); font-weight: 600; font-size: .92rem; }
  .row { display: inline-flex; align-items: center; gap: .4rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .found { list-style: none; padding: 0; display: grid; gap: .4rem; width: 100%; }
  .found li { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .7rem; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); text-align: start; }
  .found img { width: 1.9rem; }
  .found b { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bad { color: var(--red); font-size: .9rem; }
</style>
