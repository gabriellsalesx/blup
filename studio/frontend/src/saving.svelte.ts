// A small helper for "Save" buttons: busy, saved and error states.
export function saver() {
  const state = $state({ busy: false, ok: false, error: "" });
  let timer: ReturnType<typeof setTimeout> | undefined;
  return {
    state,
    async run(fn: () => Promise<unknown>) {
      state.busy = true;
      state.ok = false;
      state.error = "";
      clearTimeout(timer);
      try {
        await fn();
        state.ok = true;
        timer = setTimeout(() => (state.ok = false), 2500);
      } catch (e) {
        state.error = String(e).replace(/^(Runtime)?Error: /, "");
      } finally {
        state.busy = false;
      }
    },
  };
}
