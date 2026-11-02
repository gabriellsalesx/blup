// The window's side of aiconsent.ts: the question as a dialog over
// everything, and the answer kept through Go (SetAIConsent).
import { mount, unmount } from "svelte";
import { SetAIConsent } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import AIConsent from "./AIConsent.svelte";
import { withConsent, type Dest } from "./aiconsent";

export { ConsentCancelled, needsProvider } from "./aiconsent";

/** Asks whether the note's text may go to dest; true when accepted. */
function askAIConsent(dest: Dest, action: "translate" | "narrate"): Promise<boolean> {
  return new Promise((done) => {
    const box = mount(AIConsent, {
      target: document.body,
      props: { dest, action, onAnswer: (ok: boolean) => { unmount(box); done(ok); } },
    });
  });
}

/** Runs a translation or narration, asking first when Go says so. */
export function withAIConsent<T>(run: () => Promise<T>, action: "translate" | "narrate"): Promise<T> {
  return withConsent(run, (d) => askAIConsent(d, action), (d) => SetAIConsent(d, true));
}
