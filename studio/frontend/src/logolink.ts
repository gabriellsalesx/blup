// The logo's own link (Appearance): the same rule as the site's
// (config.ValidLogoLink in Go): an http(s) address with a host, or a path
// of the site starting with one "/". Anything else keeps the home page.
export function validLogoLink(s: string): boolean {
  if (!s || s.length > 2000 || /[\s\\<>"'`\u0000-\u001f\u007f]/.test(s)) return false;
  if (s.startsWith("/")) return !s.startsWith("//");
  try {
    const u = new URL(s);
    return (u.protocol === "https:" || u.protocol === "http:") && !!u.hostname && !u.username && !u.password && /^https?:\/\/[^/]/i.test(s);
  } catch {
    return false;
  }
}
