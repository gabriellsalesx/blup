// Sound on Linux: WebKitGTK starts GStreamer, its media engine, the first
// time a page touches audio, on the page's own thread. Measured here it
// holds the window for about half a second (two on a computer where
// GStreamer never ran), and a voice sample's ▶ was that first time.
// primeSound makes it happen while the Studio opens; after it, decoding a
// sample takes some 60 ms.
// Only on Linux: macOS and Windows play sound through the system's own
// media engine, already running.
export function primeSound() {
  if (!/Linux/.test(navigator.userAgent) || /Android/.test(navigator.userAgent)) return;
  try { new Audio().canPlayType("audio/mpeg"); } catch {}
}

// One sound output for the voice samples, made on the first ▶ (a click,
// which the web view requires to start sound).
let ctx: AudioContext | null = null;
export const soundOut = () => (ctx ??= new AudioContext());
