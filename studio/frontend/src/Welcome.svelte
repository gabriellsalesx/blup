<script lang="ts">
  import { BlupSites, BringBlupSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupAccount, BlupSite } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import CloudAccount from "./CloudAccount.svelte";
  import StepScreen from "./ui/StepScreen.svelte";
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

{#if step === "start"}
  <StepScreen image="/scenes/blog-pessoal.svg" title={h.title} lead={h.lead}>
    <div class="start-buttons">
      <button class="btn big" onclick={start}>{h.start}</button>
      <button class="btn ghost big" onclick={() => (step = "account")}>{h.account}</button>
    </div>
    <p class="note">{h.optional}</p>
  </StepScreen>
{:else if step === "account"}
  <StepScreen image="/scenes/boas-vindas-encontrados.svg" title={h.accountTitle} lead="{h.accountLead} {t.cloud.signInHelp}">
    <div class="box" class:card={account?.signedIn}><CloudAccount bind:account bare /></div>
    {#if error}<p class="bad">{error}</p>{/if}
    <p class="note">{h.optional}</p>
    {#snippet actions()}
      <button class="btn ghost" onclick={() => (step = "start")}>{t.wel.back}</button>
      <button class="link" onclick={start}>{h.instead}</button>
    {/snippet}
  </StepScreen>
{:else}
  <StepScreen image="/scenes/boas-vindas-encontrados.svg" title={sites === null ? h.looking : sites.length ? h.bringTitle : h.noneTitle} lead={sites === null ? "" : sites.length ? h.bringLead : h.noneLead}>
    {#if sites === null}
      <span class="spin"><Loader size={20} /></span>
    {:else if sites.length}
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
    {/if}
    {#if error}<p class="bad">{error}</p>{/if}
    {#snippet actions()}
      <button class="btn ghost" onclick={() => (step = "account")}>{t.wel.back}</button>
      <button class="btn" onclick={start}>{h.fresh}</button>
    {/snippet}
  </StepScreen>
{/if}

<style>
  .start-buttons { display: grid; gap: .6rem; width: min(100%, 22rem); margin-block-start: .5rem; }
  .note { color: var(--muted); font-size: .85rem; }
  .box { width: min(100%, 22rem); }
  .box.card { width: 100%; text-align: start; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .found { list-style: none; padding: 0; display: grid; gap: .4rem; width: 100%; }
  .found li { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .7rem; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); text-align: start; }
  .found img { width: 1.9rem; }
  .found b { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bad { color: var(--red); font-size: .9rem; }
</style>
