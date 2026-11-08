<script lang="ts">
  import NotebookPen from "@lucide/svelte/icons/notebook-pen";
  import Search from "@lucide/svelte/icons/search";
  import CalendarDays from "@lucide/svelte/icons/calendar-days";
  import User from "@lucide/svelte/icons/user";
  import { app, type Tab } from "../lib/app.svelte";
  import { t } from "../desk/i18n";

  const tabs = [
    { id: "notes", icon: NotebookPen },
    { id: "search", icon: Search },
    { id: "calendar", icon: CalendarDays },
    { id: "account", icon: User },
  ] satisfies { id: Tab; icon: typeof Search }[];
</script>

<nav aria-label={t.tabs.nav}>
  {#each tabs as tab (tab.id)}
    <button aria-label={t.tabs[tab.id]} aria-current={app.tab === tab.id ? "page" : undefined} onclick={() => app.select(tab.id)}>
      <span class="pill"><tab.icon size={24} /></span>
    </button>
  {/each}
</nav>

<style>
  nav { flex: none; display: grid; grid-template-columns: repeat(4, 1fr); padding: .4rem .25rem calc(.4rem + var(--sab)); background: var(--panel); border-block-start: 1px solid var(--line); }
  button { display: grid; place-items: center; color: var(--muted); min-height: 3rem; }
  .pill { display: grid; place-items: center; width: 3.6rem; height: 2.2rem; border-radius: 999px; transition: background .15s; }
  [aria-current] .pill { background: color-mix(in srgb, var(--blue) 18%, transparent); color: var(--blue-text); }
</style>
