<script lang="ts">
  import { BlupSites, BringBlupSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupSite } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";

  // The sites and vaults kept on Blup Cloud that aren't on this computer
  // (made on another computer or in Blup Notes on the phone): one tap
  // brings the notes and the settings, and they sync from then on. The
  // same as the welcome screen's, under "New site" and "New vault". Shown
  // only when there is something to bring: signed out, notes locked or
  // nothing synced, the screen stays as it was.
  let { title, onBrought }: { title: string; onBrought: (s: SiteInfo) => void } = $props();

  let items = $state<BlupSite[]>([]);
  let bringing = $state("");
  let error = $state("");
  BlupSites().then((list) => (items = (list ?? []).filter((s) => !s.dir))).catch(() => {});

  async function bring(s: BlupSite) {
    bringing = s.vault;
    error = "";
    try {
      onBrought(await BringBlupSite(s.vault));
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
    } finally {
      bringing = "";
    }
  }
</script>

{#if items.length}
  <div class="bring">
    <b>{title}</b>
    <ul>
      {#each items as s (s.vault)}
        <li>
          <img src="/blups/turma-3.svg" alt="" />
          <span class="name">{s.title}</span>
          <button class="btn" disabled={!!bringing} onclick={() => bring(s)}>
            {#if bringing === s.vault}<span class="spin"><Loader size={15} /></span>{t.wel.bringing}{:else}{t.wel.bring}{/if}
          </button>
        </li>
      {/each}
    </ul>
    {#if error}<p class="bad">{error}</p>{/if}
  </div>
{/if}

<style>
  .bring { display: grid; gap: .6rem; width: min(28rem, 100%); margin-block-start: 1rem; padding: 1rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); text-align: start; }
  ul { list-style: none; padding: 0; display: grid; gap: .2rem; max-height: 16rem; overflow: auto; }
  li { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .65rem; padding: .4rem .5rem; }
  img { width: 1.9rem; }
  .name { font-weight: 600; font-size: .9rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .btn { display: inline-flex; align-items: center; gap: .35rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
