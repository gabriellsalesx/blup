<script lang="ts">
  import { toasts, dismiss } from "./toast.svelte";
  import Check from "@lucide/svelte/icons/circle-check";
  import Alert from "@lucide/svelte/icons/circle-alert";
  import Info from "@lucide/svelte/icons/info";
  import X from "@lucide/svelte/icons/x";
  import { t } from "./i18n";
</script>

<div class="toaster" aria-live="polite">
  {#each toasts as n (n.id)}
    <div class="toast" data-kind={n.kind} role={n.kind === "error" ? "alert" : "status"}>
      <span class="ico">{#if n.kind === "ok"}<Check size={17} />{:else if n.kind === "error"}<Alert size={17} />{:else}<Info size={17} />{/if}</span>
      <span class="text">{n.text}</span>
      {#if n.action}<button class="act" onclick={() => { n.action!.run(); dismiss(n.id); }}>{n.action.label}</button>{/if}
      <button class="x" title={t.attach.close} aria-label={t.attach.close} onclick={() => dismiss(n.id)}><X size={14} /></button>
    </div>
  {/each}
</div>

<style>
  .toaster { position: fixed; z-index: 90; inset-block-end: 1.25rem; inset-inline: 0; display: grid; justify-items: center; gap: .5rem; pointer-events: none; }
  .toast { pointer-events: auto; display: flex; align-items: center; gap: .6rem; max-width: min(34rem, calc(100% - 2rem)); padding: .6rem .6rem .6rem .85rem; border: 1px solid var(--line); border-radius: .7rem; background: var(--panel); box-shadow: 0 14px 40px rgb(0 0 0 / .3); font-size: .875rem; color: var(--text); animation: up .18s ease-out; }
  .ico { display: grid; flex: none; }
  [data-kind="ok"] .ico { color: #16a34a; }
  [data-kind="error"] .ico { color: var(--red); }
  [data-kind="info"] .ico { color: var(--blue); }
  .text { flex: 1; }
  .act { flex: none; padding: .3rem .65rem; border-radius: .4rem; font-weight: 700; color: var(--blue); }
  .act:hover { background: var(--soft); }
  .x { display: grid; place-items: center; flex: none; padding: .3rem; border-radius: .35rem; color: var(--muted); }
  .x:hover { background: var(--soft); color: var(--text); }
  @keyframes up { from { opacity: 0; transform: translateY(.5rem); } }
</style>
