// What the local translation's worker (public/bergamot/worker.js) answers
// to a job: its id and the translated texts, or its id and an error.
export type TranslateReply = { id: number; out: string[] } | { id: number; error: string };

// translateReply reads a worker's message, or null when it is not one.
export function translateReply(d: unknown): TranslateReply | null {
  if (typeof d !== "object" || d === null) return null;
  const { id, out, error } = d as Record<string, unknown>;
  if (typeof id !== "number" || !Number.isSafeInteger(id) || id <= 0) return null;
  if (typeof error === "string") return { id, error };
  if (Array.isArray(out) && out.every((s) => typeof s === "string")) return { id, out };
  return null;
}
