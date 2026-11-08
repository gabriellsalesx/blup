// What the keypad says after a wrong PIN, in the app's language (i18n.ts).
import { fmt, t } from "../desk/i18n";

export const pinError = (wait: number) => (wait ? fmt(t.pin.wait, { s: wait }) : t.pin.wrong);
