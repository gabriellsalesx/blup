<script lang="ts">
  import { Sites } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import { t } from "./i18n";

  let { onOpen }: { onOpen: (s: SiteInfo) => void } = $props();
  let sites = $state<SiteInfo[]>([]);
  let loaded = $state(false);
  let error = $state("");

  Sites()
    .then((list) => (sites = list ?? []))
    .catch((e) => (error = String(e)))
    .finally(() => (loaded = true));
</script>

<main class="home">
  <header>
    <img src="/blup.svg" alt="" width="40" height="40" />
    <h1>Blup Studio</h1>
  </header>
  <section>
    <h2>{t.sites}</h2>
    <p class="muted">{t.sitesHelp}</p>
    {#if error}
      <p class="error">{t.error}: {error}</p>
    {:else if loaded && sites.length === 0}
      <p class="empty muted">{t.noSites}</p>
    {/if}
    <ul>
      {#each sites as s (s.dir)}
        <li>
          <button class="site" onclick={() => onOpen(s)}>
            <span class="kind">{s.model === "bento" ? t.product : t.blog}</span>
            <strong>{s.title}</strong>
            <span class="path muted">{s.folder}</span>
            <span class="go">{t.open} →</span>
          </button>
        </li>
      {/each}
    </ul>
  </section>
</main>

<style>
  .home { height: 100%; overflow: auto; padding: 3rem clamp(1.5rem, 6vw, 5rem); }
  header { display: flex; align-items: center; gap: .75rem; margin-block-end: 3rem; }
  h1 { font-size: 1.25rem; font-weight: 700; }
  h2 { font-size: 1.75rem; font-weight: 700; letter-spacing: -.02em; }
  section > p { margin-block: .25rem 1.5rem; }
  ul { list-style: none; padding: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr)); gap: 1rem; }
  .site { width: 100%; text-align: start; display: grid; gap: .35rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); transition: border-color .15s, transform .15s; }
  .site:hover { border-color: var(--muted); transform: translateY(-1px); }
  .site strong { font-size: 1.05rem; }
  .kind { font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; color: var(--blue); }
  .path { font-size: .8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; direction: rtl; text-align: left; }
  .go { margin-block-start: .5rem; font-size: .85rem; font-weight: 600; }
  .empty { padding: 2rem; border: 1px dashed var(--line); border-radius: .75rem; }
  .error { color: var(--red); }
</style>
