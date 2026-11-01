// One sound output for the Studio's voice samples, ready before the first
// ▶. On Linux (WebKitGTK) the first sound of a run loads the MP3 decoder
// and opens the sound output, and the window stalls meanwhile; done right
// after the settings open, the first ▶ plays at once.
let ctx: AudioContext | null = null;
let warm: Promise<void> | null = null;

export const soundOut = () => (ctx ??= new AudioContext());

// warmSound decodes a sample once (and wakes the output when the web view
// allows it without a click).
export function warmSound(sample: string) {
  warm ??= (async () => {
    const out = soundOut();
    const res = await fetch(sample);
    await out.decodeAudioData(await res.arrayBuffer());
    await out.resume().catch(() => {});
  })().catch(() => { warm = null; });
  return warm;
}
