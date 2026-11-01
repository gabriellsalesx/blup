// A module loaded the first time it is asked for, then kept: lazy(() =>
// import("./X.svelte")) for a view the Studio may never open in a session.
export function lazy<T>(load: () => Promise<T>): () => Promise<T> {
  let p: Promise<T> | undefined;
  return () => (p ??= load());
}
