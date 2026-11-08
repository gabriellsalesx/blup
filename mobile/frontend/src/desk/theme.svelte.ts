// The phone follows the system's look.
const media = matchMedia("(prefers-color-scheme: dark)");
export const scheme = $state({ dark: media.matches });
media.addEventListener("change", (e) => (scheme.dark = e.matches));
