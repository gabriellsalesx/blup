<script lang="ts">
  import { SettingsService, NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import { pinError } from "../lib/calendar-text";
  import { t } from "../desk/i18n";
  import PinPad from "./PinPad.svelte";

  let error = $state("");
  async function open(pin: string) {
    try {
      const res = await SettingsService.CheckPin(pin);
      if (res.ok) {
        app.locked = false;
        return;
      }
      NotesService.HapticStrong();
      error = pinError(res.wait);
    } catch (e) { error = message(e); }
  }
</script>

<!-- Until the app knows whether there is a PIN, nothing of the notes shows.
     Above every layer of the app (the editor's menu is at 70), and the note
     behind loses the focus, so a keyboard cannot type into it. -->
<div class="lock" {@attach () => { (document.activeElement as HTMLElement | null)?.blur(); }}>
  <div class="cover"></div>
  {#if app.lockKnown}<PinPad title={t.pin.enter} {error} onpin={open} />{/if}
</div>

<style>
  .lock { position: fixed; inset: 0; z-index: 100; }
  .cover { position: fixed; inset: 0; z-index: 40; background: var(--bg); }
</style>
