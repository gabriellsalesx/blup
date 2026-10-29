// A small helper for "Save" buttons: busy and error states, and a toast
// when it worked (ok: its text; false for none; default "Saved").
import { toast } from "./toast.svelte";
import { t } from "./i18n";

export function saver() {
  const state = $state({ busy: false, ok: false, error: "" });
  return {
    state,
    async run(fn: () => Promise<unknown>, ok?: string | false | (() => string | false)) {
      state.busy = true;
      state.ok = false;
      state.error = "";
      try {
        await fn();
        state.ok = true;
        const text = typeof ok === "function" ? ok() : ok;
        if (text !== false) toast(text ?? t.toast.saved);
      } catch (e) {
        state.error = String(e).replace(/^(Runtime)?Error: /, "");
      } finally {
        state.busy = false;
      }
    },
  };
}
