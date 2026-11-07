<script lang="ts">
  import Bug from "@lucide/svelte/icons/bug";
  import { t } from "./i18n";

  // The question, once: may the Studio send error reports to Blup? What
  // goes, what never goes, and that it changes in the settings. Escape or
  // a click outside means "not now": it is asked again next time.
  let { onAnswer }: { onAnswer: (yes: boolean | null) => void } = $props();
  const e = t.errs;
</script>

<svelte:window onkeydown={(ev) => ev.key === "Escape" && onAnswer(null)} />
<div class="shade" role="presentation" onclick={() => onAnswer(null)}></div>
<div class="dialog" role="alertdialog" aria-modal="true" aria-labelledby="err-title" aria-describedby="err-what">
  <header class="head">
    <Bug size={20} />
    <h2 id="err-title">{e.title}</h2>
  </header>
  <p id="err-what">{e.what}</p>
  <ul>
    <li>{e.only}</li>
    <li>{e.never}</li>
    <li>{e.change}</li>
  </ul>
  <div class="actions">
    <button class="btn ghost" onclick={() => onAnswer(false)}>{e.no}</button>
    <button class="btn" {@attach (el) => el.focus()} onclick={() => onAnswer(true)}>{e.yes}</button>
  </div>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 80; background: rgb(0 0 0 / .5); }
  .dialog { position: fixed; z-index: 81; inset: 0; margin: auto; width: min(30rem, calc(100% - 2rem)); height: fit-content; max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: .85rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); color: var(--text); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .head { display: flex; align-items: center; gap: .6rem; color: var(--blue); }
  .head h2 { flex: 1; font-size: 1.05rem; font-weight: 700; color: var(--text); }
  p { margin: 0; font-size: .92rem; line-height: 1.45; }
  ul { margin: 0; padding-inline-start: 1.1rem; display: grid; gap: .3rem; font-size: .85rem; color: var(--muted); }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; flex-wrap: wrap; }
</style>
