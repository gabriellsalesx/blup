// One at a time: a call waits for the one before it to end, even when that
// one failed. A save started while another is on its way would otherwise
// cross it to Go, and the older text could reach the disk last.
export function serial(run: () => Promise<void>): () => Promise<void> {
  let last: Promise<void> = Promise.resolve();
  return () => {
    const next = last.catch(() => {}).then(run);
    last = next;
    return next;
  };
}
