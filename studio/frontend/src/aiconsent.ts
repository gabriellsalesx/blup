// Sending a note's text to an AI in the cloud (OpenRouter or Blup Cloud)
// needs the person's yes, once for each. The Go side refuses without it
// ("ai consent needed: <dest>"); here the window asks and tries again.
// Kept free of the window's bindings so it can be tested on its own.

export type Dest = "openrouter" | "blup";

const msg = (e: unknown) => String((e as { message?: unknown })?.message ?? e);

/** The destination the person still has to accept, from the Go error. */
export function consentDest(e: unknown): Dest | null {
  const m = /ai consent needed: (openrouter|blup)\b/.exec(msg(e));
  return m ? (m[1] as Dest) : null;
}

/** Nobody was chosen to translate (the Studio never picks the cloud alone). */
export const needsProvider = (e: unknown) => msg(e).includes("no AI provider chosen");

/** The person said no: nothing was sent, nothing to report. */
export class ConsentCancelled extends Error {
  constructor() { super("cancelled"); this.name = "ConsentCancelled"; }
}

/**
 * Runs run; when Go asks for consent, asks the person (ask), keeps the
 * yes (accept) and runs it again. A no throws ConsentCancelled.
 */
export async function withConsent<T>(run: () => Promise<T>, ask: (d: Dest) => Promise<boolean>, accept: (d: Dest) => Promise<void>): Promise<T> {
  for (let tries = 0; ; tries++) {
    try {
      return await run();
    } catch (e) {
      const d = consentDest(e);
      if (!d || tries >= 2) throw e;
      if (!(await ask(d))) throw new ConsentCancelled();
      await accept(d);
    }
  }
}
