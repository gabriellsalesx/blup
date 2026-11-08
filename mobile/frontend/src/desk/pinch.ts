// Two fingers on a board: they move it and zoom it. onStart lets go of what
// the first finger had begun.
export function pinch(el: HTMLElement, on: { start: () => void; move: (scale: number, x: number, y: number, dx: number, dy: number) => void }) {
  let last: { d: number; x: number; y: number } | null = null;
  const read = (e: TouchEvent) => {
    const [a, b] = [e.touches[0], e.touches[1]];
    return { d: Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY), x: (a.clientX + b.clientX) / 2, y: (a.clientY + b.clientY) / 2 };
  };
  const down = (e: TouchEvent) => { if (e.touches.length === 2) { last = read(e); on.start(); } };
  const move = (e: TouchEvent) => {
    if (e.touches.length !== 2 || !last) return;
    e.preventDefault();
    const now = read(e);
    on.move(now.d / Math.max(last.d, 1), now.x, now.y, now.x - last.x, now.y - last.y);
    last = now;
  };
  const up = (e: TouchEvent) => { if (e.touches.length < 2) last = null; };
  el.addEventListener("touchstart", down, { capture: true, passive: true });
  el.addEventListener("touchmove", move, { capture: true, passive: false });
  el.addEventListener("touchend", up, { capture: true });
  el.addEventListener("touchcancel", up, { capture: true });
  return { destroy() {
    el.removeEventListener("touchstart", down, { capture: true });
    el.removeEventListener("touchmove", move, { capture: true });
    el.removeEventListener("touchend", up, { capture: true });
    el.removeEventListener("touchcancel", up, { capture: true });
  } };
}
