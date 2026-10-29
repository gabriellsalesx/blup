<script lang="ts">
  import Home from "./Home.svelte";
  import Workspace from "./Workspace.svelte";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

  // The open site, or null on the list of sites.
  let site = $state<SiteInfo | null>(null);
</script>

{#if site}
  <Workspace {site} onBack={() => (site = null)} />
{:else}
  <Home onOpen={(s) => (site = s)} />
{/if}
