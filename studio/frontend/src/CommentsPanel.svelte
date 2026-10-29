<script lang="ts">
  import { SetFeedback, SetupFeedback, FeedbackState, Connections, ConnectGitHub, OpenKeyPage, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings, FeedbackSetup } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Switch from "./Switch.svelte";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import Check from "@lucide/svelte/icons/check";
  import External from "@lucide/svelte/icons/external-link";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";

  // Comments, in three steps the first time: the GitHub key, the public
  // repository Blup creates, and the login app installed on it (so people
  // comment without leaving the site). After that, a switch.
  let { settings, reload }: { settings: Settings; reload: () => void } = $props();
  const c = t.com;
  let on = $state(settings.feedback);
  let github = $state<string | null>(null); // the GitHub login, "" when not connected
  let key = $state("");
  let busy = $state(false);
  let error = $state("");
  let setup = $state<FeedbackSetup | null>(null);
  const s = saver();
  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  Connections().then((x) => (github = x.github ?? "")).catch(() => (github = ""));
  if (settings.feedbackRepo) FeedbackState().then((x) => (setup = x)).catch(() => {});

  async function run(fn: () => Promise<void>) {
    busy = true; error = "";
    try { await fn(); } catch (e) { error = msg(e); } finally { busy = false; }
  }
  const connect = () => run(async () => { await ConnectGitHub(key); key = ""; github = (await Connections()).github ?? ""; toast(t.toast.connected); });
  const create = () => run(async () => { setup = await SetupFeedback(); on = true; reload(); toast(t.toast.commentsOn); });
  const checkApp = () => run(async () => { setup = await FeedbackState(); });
  const repo = $derived(setup?.repo || settings.feedbackRepo);
</script>

<div class="panel">
  <h1>{c.title}</h1>
  <p class="lead">{settings.model === "bento" ? c.leadBento : c.lead}</p>

  {#if !repo}
    <section class="card">
      <h2>{c.firstTime}</h2>
      <ol class="steps">
        <li class:done={!!github}>
          <b>{c.step1}</b>
          {#if github}<span class="ok"><Check size={14} /> {c.connectedAs.replace("{user}", github)}</span>
          {:else if github === ""}
            <p>{c.step1Help} <button class="link" onclick={() => OpenKeyPage("github")}>{c.openForm} <External size={13} /></button></p>
            <div class="row">
              <input type="password" bind:value={key} placeholder={c.pasteKey} autocomplete="off" />
              <button class="btn" disabled={busy || !key.trim()} onclick={connect}>{c.connect}</button>
            </div>
          {/if}
        </li>
        <li>
          <b>{c.step2}</b>
          <p>{c.step2Help}</p>
          <button class="btn" disabled={busy || !github} onclick={create}>{#if busy}<span class="spin"><Loader size={15} /></span>{/if}{c.create}</button>
        </li>
      </ol>
      {#if error}<p class="bad">{error}</p>{/if}
    </section>
  {:else}
    <section class="card">
      <Switch label={c.on} bind:checked={on} disabled={s.state.busy} onchange={(v) => s.run(async () => { await SetFeedback(v); reload(); }, v ? t.toast.commentsOn : t.toast.commentsOff)} />
      <p>{c.repo}: <button class="link" onclick={() => OpenURL(`https://github.com/${repo}`)}>{repo} <External size={13} /></button></p>
      <div class="actions"><Saved state={s.state} /></div>
    </section>

    <section class="card">
      <h2>{c.appTitle}</h2>
      {#if setup?.app}
        <p class="ok"><Check size={15} /> {c.appOk}</p>
      {:else}
        <p>{c.appHelp}</p>
        <ol class="steps small">
          <li>{c.app1}</li>
          <li>{c.app2.replace("{repo}", repo.split("/")[1] ?? repo)}</li>
          <li>{c.app3}</li>
        </ol>
        <div class="actions">
          <button class="btn" onclick={() => setup && OpenURL(setup.installURL)} disabled={!setup}>{c.install} <External size={14} /></button>
          <button class="btn ghost" disabled={busy} onclick={checkApp}>{busy ? c.checking : c.installed}</button>
        </div>
      {/if}
      {#if error}<p class="bad">{error}</p>{/if}
    </section>
    <p class="muted">{c.republish}</p>
  {/if}
</div>

<style>
  .steps { display: grid; gap: 1.1rem; padding-inline-start: 1.25rem; }
  .steps li { display: grid; gap: .4rem; }
  .steps li p { font-size: .9rem; color: var(--muted); }
  .steps li > .btn { justify-self: start; }
  .steps.small { gap: .35rem; font-size: .9rem; }
  .steps.small li { display: list-item; }
  .ok { display: inline-flex; align-items: center; gap: .3rem; color: #16a34a; font-size: .9rem; }
  .link { display: inline-flex; align-items: center; gap: .25rem; color: var(--blue); }
  .link:hover { text-decoration: underline; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
