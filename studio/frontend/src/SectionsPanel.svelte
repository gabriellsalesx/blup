<script lang="ts">
  import { AddSection } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";

  let { settings, reload, onTree }: { settings: Settings; reload: () => void; onTree: (n: Node) => void } = $props();
  const s = saver();
  const label = (k: string) => (t.sec as Record<string, string>)[k];
</script>

<div class="panel">
  <h1>{t.sec.title}</h1>
  <p class="lead">{t.sec.lead}</p>
  {#each settings.sections ?? [] as sec (sec.kind)}
    <section class="card section">
      <div class="field"><span>{label(sec.kind)}</span><small>{label(sec.kind + "Help")}</small></div>
      {#if sec.present}
        <span class="badge on">✓ {t.sec.present} · {sec.folder}</span>
      {:else}
        <button class="btn ghost" disabled={s.state.busy} onclick={() => s.run(async () => { onTree(await AddSection(sec.kind)); reload(); })}>+ {t.sec.add}</button>
      {/if}
    </section>
  {/each}
  <p class="muted hint">{t.sec.removeHint}</p>
  <div class="actions"><Saved state={s.state} /></div>
</div>

<style>
  .section { grid-template-columns: minmax(0, 1fr) auto; align-items: center; }
  .hint { font-size: .85rem; }
</style>
