<script lang="ts">
  import Home from "./Home.svelte";
  import Workspace from "./Workspace.svelte";
  import Onboarding from "./Onboarding.svelte";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

  // The open site, the new site screens, or the list of sites.
  let site = $state<SiteInfo | null>(null);
  let creating = $state(false);
  let hasSites = $state(true);
</script>

{#if site}
  {#key site.dir}
    <Workspace {site} onBack={() => (site = null)} onSwitch={(s) => (site = s)} />
  {/key}
{:else if creating}
  <Onboarding canCancel={hasSites} onCancel={() => (creating = false)} onDone={(s) => { creating = false; site = s; }} />
{:else}
  <Home onOpen={(s) => (site = s)} onNew={() => (creating = true)} onEmpty={() => { hasSites = false; creating = true; }} />
{/if}
