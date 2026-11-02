// The own domain on Blup Cloud: how each status the cloud sends shows, and
// how often the screen asks again while it is not active.

// POLL_MS: once a minute while the screen is open and the domain is not
// active (the cloud checks the DNS on each ask); no short polling.
export const POLL_MS = 60_000;

// statusKind is the color of a status: "good" (active), "warn" (waiting
// for the DNS or the setup), "bad" (error) or "muted" (unknown).
export function statusKind(status: string | undefined | null): "good" | "warn" | "bad" | "muted" {
  switch (status) {
    case "active": return "good";
    case "pending_ownership":
    case "pending": return "warn";
    case "error": return "bad";
    default: return "muted";
  }
}
