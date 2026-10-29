<script lang="ts">
  import { SetPassword } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";

  let { settings, reload }: { settings: Settings; reload: () => void } = $props();
  let pw = $state("");
  let show = $state(false);
  const s = saver();
</script>

<div class="panel">
  <h1>{t.pw.title}</h1>
  <p class="lead">{t.pw.lead}</p>
  <section class="card">
    <span class="badge" class:on={settings.hasPassword}>{settings.hasPassword ? "✓ " + t.pw.set : t.pw.notSet}</span>
    <label class="field"><span>{t.pw.label}</span>
      <div class="row">
        <input type={show ? "text" : "password"} bind:value={pw} autocomplete="new-password" />
        <button class="btn ghost" onclick={() => (show = !show)}>{t.pw.show}</button>
      </div>
    </label>
    <div class="actions">
      <Saved state={s.state} />
      {#if settings.hasPassword}<button class="btn ghost" disabled={s.state.busy} onclick={() => s.run(async () => { await SetPassword(""); reload(); })}>{t.remove}</button>{/if}
      <button class="btn" disabled={s.state.busy || !pw} onclick={() => s.run(async () => { await SetPassword(pw); pw = ""; reload(); })}>{t.save}</button>
    </div>
  </section>
</div>
