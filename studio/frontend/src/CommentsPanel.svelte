<script lang="ts">
  import { SetFeedback } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Switch from "./Switch.svelte";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";

  let { settings, reload }: { settings: Settings; reload: () => void } = $props();
  let on = $state(settings.feedback);
  const s = saver();
</script>

<div class="panel">
  <h1>{t.com.title}</h1>
  <p class="lead">{t.com.lead}</p>
  <section class="card">
    <Switch label={t.com.on} bind:checked={on} disabled={!settings.feedbackRepo || s.state.busy} onchange={(v) => s.run(async () => { await SetFeedback(v); reload(); })} />
    {#if settings.feedbackRepo}
      <p>{t.com.repo}: <b>{settings.feedbackRepo}</b></p>
    {:else}
      <p>{t.com.setup}</p>
    {/if}
    <div class="actions"><Saved state={s.state} /></div>
  </section>
</div>
