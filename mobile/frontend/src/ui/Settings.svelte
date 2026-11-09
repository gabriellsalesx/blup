<script lang="ts">
  import Bell from "@lucide/svelte/icons/bell";
  import CalendarDays from "@lucide/svelte/icons/calendar-days";
  import Lock from "@lucide/svelte/icons/lock";
  import Bug from "@lucide/svelte/icons/bug";
  import { startPageReports } from "../lib/errorreports";
  import { NotesService, SettingsService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import { pinError } from "../lib/calendar-text";
  import { t } from "../desk/i18n";
  import PinPad from "./PinPad.svelte";
  import Switch from "./Switch.svelte";

  // Relatórios de erro (errors.go): a mesma resposta da pergunta do início.
  // A resposta fica só no app (app.reportsAnswer): a pergunta do início e
  // esta chave mostram sempre a mesma.
  const reportsOn = $derived(app.reportsAnswer === "on");
  async function reports(on: boolean) {
    const before = app.reportsAnswer;
    app.reportsAnswer = on ? "on" : "off";
    try {
      const r = await SettingsService.SetErrorReports(on);
      startPageReports(r);
      app.reportsAnswer = r.answer;
      app.say({ text: on ? t.settings.reportsOn : t.settings.reportsOff });
    } catch (e) { app.reportsAnswer = before; app.say({ text: message(e) }); }
  }
  async function notifications(on: boolean) {
    try {
      await SettingsService.SetNotifications(on);
      app.notifications = on;
    } catch (e) { app.say({ text: message(e) }); }
  }

  // Creating a PIN asks it twice; taking it off asks the current one.
  let first = $state("");
  let error = $state("");
  function startPin() {
    first = error = "";
    app.go({ panel: "pin" });
  }
  async function onpin(pin: string) {
    try {
      if (app.pinOn) {
        const res = await SettingsService.RemovePin(pin);
        if (res.ok) done(false, t.pin.removed);
        else { NotesService.HapticStrong(); error = pinError(res.wait); }
      } else if (!first) {
        first = pin;
        error = "";
      } else if (pin !== first) {
        first = "";
        error = t.pin.mismatch;
      } else {
        await SettingsService.SetPin(pin);
        done(true, t.pin.locked);
      }
    } catch (e) { error = message(e); }
  }
  function done(on: boolean, text: string) {
    app.pinOn = on;
    app.back();
    app.say({ text });
  }
  const title = $derived(app.pinOn ? t.pin.toRemove : first ? t.pin.repeat : t.pin.create);
</script>

<div class="row"><span class="ico"><CalendarDays size={22} /></span><Switch label={t.settings.holidays} hint={t.settings.holidaysHint} checked={app.showHolidays} onchange={(on) => app.setHolidays(on)} /></div>
<div class="row"><span class="ico"><Bell size={22} /></span><Switch label={t.settings.notify} hint={t.settings.notifyHint} checked={app.notifications} onchange={notifications} /></div>
<div class="row"><span class="ico"><Lock size={22} /></span><Switch label={t.settings.lock} hint={t.settings.lockHint} checked={app.pinOn} onchange={startPin} /></div>
<div class="row"><span class="ico"><Bug size={22} /></span><Switch label={t.settings.reports} hint={t.settings.reportsHint} checked={reportsOn} onchange={reports} /></div>

{#if app.place.panel === "pin"}
  <PinPad {title} {error} {onpin} oncancel={() => app.back()} hint={app.pinOn ? "" : t.pin.hint} />
{/if}

<style>
  .row { display: flex; align-items: center; gap: 1.1rem; min-height: 4rem; padding: .5rem 1.25rem; }
  .ico { flex: none; display: grid; place-items: center; width: 1.6rem; color: var(--muted); }
  .row :global(button) { flex: 1; min-width: 0; font-size: 1rem; font-weight: 500; }
</style>
