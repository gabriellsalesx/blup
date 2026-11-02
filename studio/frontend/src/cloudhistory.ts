// Deleting the old versions of a vault on Blup Cloud: the sizes as the
// settings show them, and the account's use after a purge.

// bytes writes a size like the settings' usage: MB below 1 GB, else GB.
export const bytes = (n: number) => (n < 1 << 30 ? `${Math.max(0, n / (1 << 20)).toFixed(n < 10 << 20 ? 1 : 0)} MB` : `${(n / (1 << 30)).toFixed(1)} GB`);

type Use = { storageBytes: number; historyBytes: number | null };

// afterPurge is the use once the cloud answered: its storage (the
// cloud's number), and the history less what was freed (never below 0;
// still unknown when it was unknown).
export function afterPurge<T extends Use>(account: T, r: { freedBytes: number; storageBytes: number }): T {
  const h = account.historyBytes;
  return { ...account, storageBytes: Math.max(0, r.storageBytes), historyBytes: h == null ? null : Math.max(0, h - Math.max(0, r.freedBytes)) };
}

type Storage = { signedIn: boolean; storageBytes: number; storageLimit: number; graceUntil: string; graceLimit: number };

// overLimit is the storage above the plan's, for the banner: what is
// used, the plan's limit, and while a grace runs (after a smaller plan)
// its last day and the previous plan's limit. null when within the limit.
export function overLimit(a: Storage | null | undefined, now = new Date()): { used: number; limit: number; graceUntil: Date | null; graceLimit: number } | null {
  if (!a?.signedIn || !(a.storageLimit > 0) || a.storageBytes <= a.storageLimit) return null;
  const until = a.graceUntil ? new Date(a.graceUntil) : null;
  const grace = until && !Number.isNaN(until.getTime()) && until > now && a.graceLimit > 0 ? until : null;
  return { used: a.storageBytes, limit: a.storageLimit, graceUntil: grace, graceLimit: grace ? a.graceLimit : 0 };
}
