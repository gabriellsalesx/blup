// The platforms a Pulse project can be made for. The platform only picks
// how the panel explains the setup: every SDK sends the same envelopes.
// `tested` marks the SDKs already checked against Blup Pulse (a real
// error from the official SDK reaches it grouped, with its stack, and
// without personal data; the JavaScript family and the server languages
// on 05/11/2026, see docs/spec/blup-pulse.md); the others use the same
// protocol and are shown as being tested.
export const platformGroups = ["popular", "browser", "server", "mobile", "desktop"] as const;
export type PlatformGroup = (typeof platformGroups)[number];

export const platforms = [
	{ id: "javascript", label: "JavaScript", groups: ["popular", "browser"], tested: true },
	{ id: "react", label: "React", groups: ["popular", "browser"], tested: true },
	{ id: "nextjs", label: "Next.js", groups: ["popular", "browser", "server"], tested: true },
	{ id: "vue", label: "Vue", groups: ["browser"], tested: true },
	{ id: "nuxt", label: "Nuxt", groups: ["browser", "server"], tested: true },
	{ id: "angular", label: "Angular", groups: ["browser"], tested: true },
	{ id: "svelte", label: "Svelte", groups: ["browser"], tested: true },
	{ id: "sveltekit", label: "SvelteKit", groups: ["browser", "server"], tested: true },
	{ id: "astro", label: "Astro", groups: ["browser"], tested: true },
	{ id: "node", label: "Node.js", groups: ["popular", "server"], tested: true },
	{ id: "express", label: "Express", groups: ["popular", "server"], tested: true },
	{ id: "nestjs", label: "NestJS", groups: ["server"], tested: true },
	{ id: "hono", label: "Hono", groups: ["server"], tested: true },
	{ id: "bun", label: "Bun", groups: ["server"], tested: true },
	{ id: "deno", label: "Deno", groups: ["server"], tested: true },
	{ id: "python", label: "Python", groups: ["popular", "server"], tested: true },
	{ id: "django", label: "Django", groups: ["popular", "server"], tested: true },
	{ id: "flask", label: "Flask", groups: ["server"], tested: true },
	{ id: "fastapi", label: "FastAPI", groups: ["server"], tested: true },
	{ id: "go", label: "Go", groups: ["popular", "server", "desktop"], tested: true },
	{ id: "php", label: "PHP", groups: ["popular", "server"], tested: true },
	{ id: "laravel", label: "Laravel", groups: ["popular", "server"], tested: true },
	{ id: "symfony", label: "Symfony", groups: ["server"], tested: true },
	{ id: "ruby", label: "Ruby", groups: ["server"], tested: true },
	{ id: "rails", label: "Rails", groups: ["server"], tested: true },
	{ id: "java", label: "Java", groups: ["popular", "server"], tested: true },
	{ id: "spring", label: "Spring Boot", groups: ["server"], tested: true },
	{ id: "dotnet", label: "C# / .NET", groups: ["server", "desktop"], tested: true },
	{ id: "elixir", label: "Elixir", groups: ["server"], tested: true },
	{ id: "rust", label: "Rust", groups: ["server", "desktop"], tested: true },
	{ id: "react-native", label: "React Native", groups: ["popular", "mobile"], tested: false },
	{ id: "flutter", label: "Flutter", groups: ["mobile", "desktop"], tested: false },
	{ id: "android", label: "Android", groups: ["mobile"], tested: false },
	{ id: "ios", label: "iOS", groups: ["mobile"], tested: false },
	{ id: "electron", label: "Electron", groups: ["desktop"], tested: false },
	{ id: "other", label: "Other", groups: [], tested: false },
] as const satisfies readonly { id: string; label: string; groups: readonly PlatformGroup[]; tested: boolean }[];

export type PlatformId = (typeof platforms)[number]["id"];
export const platformIds = platforms.map((p) => p.id) as [PlatformId, ...PlatformId[]];
