<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { BlupDomain as LoadDomain, SetBlupDomain, RemoveBlupDomain, OpenBlupCloud } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupDomain } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Check from "@lucide/svelte/icons/check";
  import Copy from "@lucide/svelte/icons/copy";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import Refresh from "@lucide/svelte/icons/refresh-cw";
  import { Clipboard } from "@wailsio/runtime";
  import { t } from "./i18n";
  import { statusKind, polls, graceDate, POLL_MS } from "./blupdomain";

  // The site's own domain on Blup Cloud (Max and Ultra). Whether the plan
  // includes it and where the domain stands come from the cloud; the
  // records to create are shown as the cloud sends them.
  let { slug }: { slug: string } = $props();
  const b = t.bdom;

  let info = $state<BlupDomain | null>(null);
  let input = $state("");
  let busy = $state(false);
  let error = $state("");
  let askRemove = $state(false);
  let copied = $state(-1);

  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  async function run(fn: () => Promise<BlupDomain>) {
    busy = true; error = "";
    try { info = await fn(); askRemove = false; } catch (e) { error = msg(e); } finally { busy = false; }
  }
  const load = () => run(() => LoadDomain());
  const link = () => run(() => SetBlupDomain(input.trim()));
  const remove = () => run(async () => { const r = await RemoveBlupDomain(); input = ""; return r; });
  async function copy(i: number, v: string) {
    await Clipboard.SetText(v);
    copied = i;
    setTimeout(() => (copied = -1), 1500);
  }
  load();

  const domain = $derived(info?.domain ?? null);
  const kind = $derived(statusKind(domain?.status));
  // While the domain is on its way, the cloud is asked again once a
  // minute (it checks the DNS); nothing runs once it is active or off.
  const waiting = $derived(polls(domain?.status));
  // Kept after a downgrade: the day it stops.
  const grace = $derived(domain?.status === "active" ? graceDate(domain.disableAt, t.locale) : "");
  $effect(() => {
    if (!waiting) return;
    const id = setInterval(() => { if (!busy) LoadDomain().then((r) => (info = r)).catch(() => {}); }, POLL_MS);
    return () => clearInterval(id);
  });
</script>

<section class="card">
  <h2>{b.title}</h2>
  <p>{b.help.replace("{slug}", slug)}</p>
  {#if !info}
    {#if error}<p class="bad">{error}</p>{:else}<p class="muted"><span class="spin"><Loader size={14} /></span> {b.checking}</p>{/if}
  {:else if !info.available && !domain}
    <p class="box">{b.paid}</p>
    <div class="actions"><button class="btn" onclick={() => OpenBlupCloud(Page.PagePlan)}>{b.plans}</button></div>
  {:else if !info.published}
    <p class="muted">{b.unpublished}</p>
  {:else if !domain}
    <div class="row">
      <input type="text" bind:value={input} placeholder={b.placeholder} spellcheck="false" autocomplete="off" onkeydown={(e) => e.key === "Enter" && input.trim() && link()} />
      <button class="btn" disabled={busy || !input.trim()} onclick={link}>{busy ? b.linking : b.link}</button>
    </div>
    <small class="muted">{b.inputHelp}</small>
  {:else}
    <div class="row status">
      <div class="field"><span>{domain.hostname}</span>
        <small class={kind}>{#if kind === "good"}<Check size={13} />{/if}{b.status[domain.status as keyof typeof b.status] ?? domain.status} · {b.ssl[domain.ssl as keyof typeof b.ssl] ?? domain.ssl}</small>
      </div>
      <span class="spacer"></span>
      <button class="btn ghost" disabled={busy} onclick={load}>{#if busy}<span class="spin"><Loader size={14} /></span>{:else}<Refresh size={14} />{/if}{b.check}</button>
    </div>
    {#if domain.status === "disabled"}
      <p class="box">{b.disabledHelp.replace("{domain}", domain.hostname).replace("{slug}", slug)}</p>
      {#if !info.available}<div class="actions"><button class="btn" onclick={() => OpenBlupCloud(Page.PagePlan)}>{b.plans}</button></div>{/if}
    {:else if grace}
      <p class="box">{b.grace.replace("{domain}", domain.hostname).replace("{date}", grace)}</p>
      <div class="actions"><button class="btn" onclick={() => OpenBlupCloud(Page.PagePlan)}>{b.plans}</button></div>
    {/if}
    {#if domain.status !== "active" && domain.status !== "disabled" && domain.records?.length}
      <p>{b.records}</p>
      <table class="dns">
        <thead><tr><th>{b.type}</th><th>{b.name}</th><th>{b.value}</th><th></th></tr></thead>
        <tbody>
          {#each domain.records as r, i (i)}
            <tr><td>{r.type}</td><td><code>{r.name}</code></td><td><code>{r.value}</code></td>
              <td><button class="icon-btn" title={b.copy} aria-label={b.copy} onclick={() => copy(i, r.value)}>{#if copied === i}<Check size={14} />{:else}<Copy size={14} />{/if}</button></td></tr>
          {/each}
        </tbody>
      </table>
      {#if info.apex}<p class="box">{b.apex.replaceAll("{domain}", domain.hostname)}</p>{/if}
      <small class="muted">{b.dnsTime} {b.auto}</small>
    {/if}
    <div class="actions">
      {#if askRemove}
        <span class="muted q">{b.removeQ.replace("{domain}", domain.hostname).replace("{slug}", slug)}</span>
        <button class="btn ghost" onclick={() => (askRemove = false)}>{b.cancel}</button>
        <button class="btn danger" disabled={busy} onclick={remove}>{b.remove}</button>
      {:else}
        <button class="btn ghost danger-text" onclick={() => (askRemove = true)}>{b.remove}</button>
      {/if}
    </div>
  {/if}
  {#if error && info}<p class="bad">{error}</p>{/if}
</section>

<style>
  .box { padding: .75rem 1rem; border-radius: .5rem; background: var(--soft); }
  .status { align-items: center; }
  .spacer { flex: 1; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .good { display: inline-flex; align-items: center; gap: .25rem; color: #16a34a; }
  .warn { color: #b45309; }
  .dns { width: 100%; border-collapse: collapse; font-size: .85rem; }
  .dns th { text-align: start; font-size: .75rem; color: var(--muted); font-weight: 600; padding: .3rem .5rem; }
  .dns td { padding: .45rem .5rem; border-block-start: 1px solid var(--line); word-break: break-all; }
  .dns code { font-size: .85rem; user-select: all; }
  .q { font-size: .85rem; }
  .danger-text { color: var(--red); }
  .btn.danger { background: var(--red); color: #fff; }
</style>
