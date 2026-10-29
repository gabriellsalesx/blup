// The social networks the Blup theme knows (theme/layouts/_partials/site/
// network.html and its icons): what an author's profile link points to.
export type Network = { key: string; name: string; prefix: string; match: string[]; icon: string };

const svg = (paths: string) => `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;

export const NETWORKS: Network[] = [
  { key: "instagram", name: "Instagram", prefix: "https://instagram.com/", match: ["instagram.com"], icon: svg('<rect x="3" y="3" width="18" height="18" rx="5"/><circle cx="12" cy="12" r="4"/><path d="M17 7h.01"/>') },
  { key: "x", name: "X", prefix: "https://x.com/", match: ["x.com/", "twitter.com"], icon: svg('<path d="M3.5 3H8l12.5 18H16Z"/><path d="m20.5 3-7.2 7.63M10.7 13.37 3.5 21"/>') },
  { key: "linkedin", name: "LinkedIn", prefix: "https://linkedin.com/in/", match: ["linkedin.com"], icon: svg('<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M7 10v7M11 17v-7M11 13a3 3 0 0 1 6 0v4M7 7h.01"/>') },
  { key: "youtube", name: "YouTube", prefix: "https://youtube.com/@", match: ["youtube.com", "youtu.be"], icon: svg('<rect x="3" y="5" width="18" height="14" rx="4"/><path d="m10 9 5 3-5 3Z"/>') },
  { key: "facebook", name: "Facebook", prefix: "https://facebook.com/", match: ["facebook.com"], icon: svg('<path d="M17 3h-2a4 4 0 0 0-4 4v14M7 10h10"/>') },
  { key: "github", name: "GitHub", prefix: "https://github.com/", match: ["github.com"], icon: svg('<path d="M9 21v-3a3.6 3.6 0 0 1 .8-2.6C7.2 15.1 4 14.1 4 9.4a4.7 4.7 0 0 1 1.3-3.3A4.3 4.3 0 0 1 5.4 3S6.4 2.7 9 4a10.8 10.8 0 0 1 6 0c2.6-1.3 3.6-1 3.6-1a4.3 4.3 0 0 1 .1 3.1A4.7 4.7 0 0 1 20 9.4c0 4.7-3.2 5.7-5.8 6A3.6 3.6 0 0 1 15 18v3M9 18.2c-4 .9-4.2-2-6-2.5"/>') },
  { key: "whatsapp", name: "WhatsApp", prefix: "https://wa.me/", match: ["wa.me", "whatsapp.com"], icon: svg('<path d="M21 12a9 9 0 0 1-13.5 7.79L3 21l1.21-4.5A9 9 0 1 1 21 12Z"/><path d="m8 7 1.5 3-1.5 1.5a7 7 0 0 0 4.5 4.5l1.5-1.5 3 1.5"/>') },
  { key: "mail", name: "E-mail", prefix: "mailto:", match: ["mailto:"], icon: svg('<path d="m22 7-8.991 5.727a2 2 0 0 1-2.009 0L2 7"/><rect x="2" y="4" width="20" height="16" rx="2"/>') },
  { key: "link", name: "Site", prefix: "https://", match: [], icon: svg('<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>') },
];

// The network of a link (the plain link when none matches), like the theme.
export function networkOf(url: string): Network {
  const u = url.toLowerCase();
  return NETWORKS.find((n) => n.match.some((m) => u.includes(m))) ?? NETWORKS[NETWORKS.length - 1];
}
