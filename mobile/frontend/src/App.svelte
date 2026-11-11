<script lang="ts">
  import { app } from "./lib/app.svelte";
  import TabBar from "./ui/TabBar.svelte";
  import Notes from "./screens/Notes.svelte";
  import Search from "./screens/Search.svelte";
  import { screens } from "./lib/screens";
  import LockScreen from "./ui/LockScreen.svelte";
  import NoteLayer from "./ui/NoteLayer.svelte";
  import ErrorReportsAsk from "./ui/ErrorReportsAsk.svelte";
  import { SettingsService } from "../bindings/github.com/gabriellsalesx/blup/mobile";
  import { startPageReports } from "./lib/errorreports";

  app.start();

  // The note one step back, when it is another note than the open one; or,
  // just after going back to it, that same drawing on top until the note's
  // own editor has drawn (app.handoff): the same element, never drawn anew.
  const noteBelow = $derived.by(() => {
    const id = app.handoff ?? app.below?.note;
    return id && (id === app.handoff || id !== app.place.note) ? (app.notes.find((n) => n.id === id) ?? null) : null;
  });

  // Relatórios de erro para o Blup (errors.go): perguntado uma vez, numa
  // gaveta, depois de destrancar; com "sim", os erros da tela vão também.
  let askReports = $state(false);
  SettingsService.ErrorReports()
    .then((r) => { startPageReports(r); app.reportsAnswer = r.answer; askReports = r.answer === ""; })
    .catch(() => {});
  async function answerReports(yes: boolean | null) {
    askReports = false;
    if (yes === null) return;
    try {
      const r = await SettingsService.SetErrorReports(yes);
      startPageReports(r);
      app.reportsAnswer = r.answer;
    } catch { /* pergunta de novo na próxima vez */ }
  }
</script>

<main>
  {#if app.error}
    <p class="error">{app.error}</p>
  {:else}
    <div class="screen" inert={!!app.current || !!app.place.board}>
      {#if app.tab === "notes"}<Notes />
      {:else if app.tab === "search"}<Search />
      {:else if app.tab === "calendar"}{#await screens.calendar() then { default: Calendar }}<Calendar />{/await}
      {:else}{#await screens.account() then { default: Account }}<Account />{/await}{/if}
    </div>
    {#if !app.keyboard && !app.place.event}<TabBar />{/if}
    {#if app.current}
      <!-- The note back goes to (opened from this one, by a link or
           "Mentioned in"), drawn as it is under the open one, which slides
           away over it (NoteLayer). -->
      {#if noteBelow}{#key noteBelow.id}<div class={["below", { handoff: app.handoff === noteBelow.id }]} inert aria-hidden="true">{#await screens.editor() then { default: Editor }}<Editor note={noteBelow} still />{/await}</div>{/key}{/if}
      {#key app.noteKey}<NoteLayer>{#await screens.editor() then { default: Editor }}<Editor note={app.current} />{/await}</NoteLayer>{/key}
    {:else if app.place.board}
      {#key app.place.board}<NoteLayer>{#await screens.board() then { default: Board }}<Board path={app.place.board} />{/await}</NoteLayer>{/key}
    {/if}
  {/if}
  {#if app.toast}
    <div class="toast" role="status">
      <span>{app.toast.text}</span>
      {#if app.toast.action}<button onclick={() => { app.toast?.action?.run(); app.toast = null; }}>{app.toast.action.label}</button>{/if}
    </div>
  {/if}
  {#if askReports && !app.locked && !app.error}<ErrorReportsAsk onAnswer={answerReports} />{/if}
  {#if app.locked}<LockScreen />{/if}
</main>

<style>
  main { position: relative; display: flex; flex-direction: column; height: calc(100% - var(--kb, 0px)); background: var(--bg); }
  .screen { position: relative; flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .toast { position: fixed; inset-inline: 1rem; inset-block-end: calc(10rem + var(--sab)); z-index: 30; display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .8rem 1rem; border-radius: .7rem; background: var(--accent); color: var(--on-accent); font-size: .9rem; }
  .toast button { font-weight: 700; }
  /* Under the open note (NoteLayer's .layer is 19, its shade 18 after it). */
  .below { position: absolute; inset: 0; z-index: 18; display: flex; flex-direction: column; background: var(--bg); }
  .below.handoff { z-index: 20; }
  .error { padding: 4rem 1rem; color: var(--red); overflow-wrap: anywhere; }
</style>
