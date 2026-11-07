<script lang="ts">
  import Home from "./Home.svelte";
  import Workspace from "./Workspace.svelte";
  import Onboarding from "./Onboarding.svelte";
  import Welcome from "./Welcome.svelte";
  import VaultOnboarding from "./VaultOnboarding.svelte";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import { Sites, ErrorReports, SetErrorReports } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import ErrorReportsAsk from "./ErrorReportsAsk.svelte";
  import { startPageReports } from "./errorreports";
  import { look, setPref } from "./theme.svelte";
  import Toaster from "./Toaster.svelte";

  // The open site, the new site screens, or the list of sites.
  let site = $state<SiteInfo | null>(null);
  let creating = $state(false);
  let creatingVault = $state(false);

  // The welcome screens, the first time on a computer without sites (one
  // that already has sites skips them). Then the Studio opens on the site
  // it was on when it closed (Workspace opens the note it was on).
  let welcome = $state(false);
  let ready = $state(false);
  Sites()
    .then((list) => {
      list = list ?? [];
      if (!look.welcomed) {
        if (list.length) setPref("welcomed", true);
        else welcome = true;
      }
      if (look.lastSite) site = list.find((s) => s.dir === look.lastSite) ?? null;
    })
    .catch(() => {})
    .finally(() => (ready = true));
  $effect(() => { if (ready && (site?.dir ?? "") !== look.lastSite) setPref("lastSite", site?.dir ?? ""); });

  // Error reports to Blup (errors.go): asked once, after the welcome; the
  // page's own errors go too when the answer is yes. "Not now" (Escape)
  // asks again the next time the Studio opens.
  let askReports = $state(false);
  ErrorReports()
    .then((r) => { startPageReports(r); askReports = r.answer === ""; })
    .catch(() => {});
  async function answerReports(yes: boolean | null) {
    askReports = false;
    if (yes === null) return;
    try { startPageReports(await SetErrorReports(yes)); } catch { /* asked again next time */ }
  }
</script>

{#if !ready}
  <!-- Finding the last site. -->
{:else if welcome}
  <Welcome onStart={() => (welcome = false)} onOpen={(s) => { welcome = false; site = s; }} />
{:else if site}
  {#key site.dir}
    <Workspace {site} onBack={() => (site = null)} onSwitch={(s) => (site = s)} />
  {/key}
{:else if creatingVault}
  <VaultOnboarding onCancel={() => (creatingVault = false)} onDone={(v) => { creatingVault = false; site = v; }} />
{:else if creating}
  <Onboarding canCancel onCancel={() => (creating = false)} onDone={(s) => { creating = false; site = s; }} />
{:else}
  <Home onOpen={(s) => (site = s)} onNew={() => (creating = true)} onNewVault={() => (creatingVault = true)} />
{/if}

{#if askReports && ready && !welcome}<ErrorReportsAsk onAnswer={answerReports} />{/if}

<Toaster />
