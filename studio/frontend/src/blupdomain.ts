// The own domain on Blup Cloud: how each status the cloud sends shows, and
// how often the screen asks again while it is not active.

// POLL_MS: once a minute while the screen is open and the domain is not
// active (the cloud checks the DNS on each ask); no short polling.
export const POLL_MS = 60_000;

// statusKind is the color of a status: "good" (active), "warn" (waiting
// for the DNS or the setup), "bad" (error, or turned off after a plan
// without own domains) or "muted" (unknown).
export function statusKind(status: string | undefined | null): "good" | "warn" | "bad" | "muted" {
  switch (status) {
    case "active": return "good";
    case "pending_ownership":
    case "pending": return "warn";
    case "error":
    case "disabled": return "bad";
    default: return "muted";
  }
}

// polls: whether the screen asks the cloud again once a minute. Only while
// the domain is on its way (TXT, setup, error to fix): an active one is
// done and a turned off one waits for a new link.
export const polls = (status: string | undefined | null) => !!status && status !== "active" && status !== "disabled";

// graceDate is the day a domain kept after a downgrade stops, written for
// the person ("" when there is none or it isn't a date).
export function graceDate(disableAt: string | null | undefined, lang: string): string {
  if (!disableAt) return "";
  const d = new Date(disableAt);
  return Number.isNaN(d.getTime()) ? "" : d.toLocaleDateString(lang, { day: "numeric", month: "long", year: "numeric" });
}
