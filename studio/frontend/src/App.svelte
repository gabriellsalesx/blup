<script lang="ts">
  import Home from "./Home.svelte";
  import Workspace from "./Workspace.svelte";
  import Onboarding from "./Onboarding.svelte";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import { Sites } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import { look, setPref } from "./theme.svelte";

  // The open site, the new site screens, or the list of sites.
  let site = $state<SiteInfo | null>(null);
  let creating = $state(false);
  let hasSites = $state(true);

  // The Studio opens on the site it was on when it closed (Workspace opens
  // the note it was on).
  let ready = $state(!look.lastSite);
  if (look.lastSite) {
    Sites().then((list) => { site = (list ?? []).find((s) => s.dir === look.lastSite) ?? null; }).catch(() => {}).finally(() => (ready = true));
  }
  $effect(() => { if (ready && (site?.dir ?? "") !== look.lastSite) setPref("lastSite", site?.dir ?? ""); });
</script>

{#if !ready}
  <!-- Finding the last site. -->
{:else if site}
  {#key site.dir}
    <Workspace {site} onBack={() => (site = null)} onSwitch={(s) => (site = s)} />
  {/key}
{:else if creating}
  <Onboarding canCancel={hasSites} onCancel={() => (creating = false)} onDone={(s) => { creating = false; site = s; }} />
{:else}
  <Home onOpen={(s) => (site = s)} onNew={() => (creating = true)} onEmpty={() => { hasSites = false; creating = true; }} />
{/if}
