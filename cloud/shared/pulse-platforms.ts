// The platforms a Pulse project can be made for. The platform only picks
// how the panel explains the setup: every SDK sends the same envelopes.
// See docs/spec/blup-pulse.md for how each SDK was checked.
export const platformGroups = ["popular", "browser", "server", "mobile", "desktop"] as const;
export type PlatformGroup = (typeof platformGroups)[number];

export const platforms = [
	{ id: "javascript", label: "JavaScript", groups: ["popular", "browser"] },
	{ id: "react", label: "React", groups: ["popular", "browser"] },
	{ id: "nextjs", label: "Next.js", groups: ["popular", "browser", "server"] },
	{ id: "vue", label: "Vue", groups: ["browser"] },
	{ id: "nuxt", label: "Nuxt", groups: ["browser", "server"] },
	{ id: "angular", label: "Angular", groups: ["browser"] },
	{ id: "svelte", label: "Svelte", groups: ["browser"] },
	{ id: "sveltekit", label: "SvelteKit", groups: ["browser", "server"] },
	{ id: "astro", label: "Astro", groups: ["browser"] },
	{ id: "node", label: "Node.js", groups: ["popular", "server"] },
	{ id: "express", label: "Express", groups: ["popular", "server"] },
	{ id: "nestjs", label: "NestJS", groups: ["server"] },
	{ id: "hono", label: "Hono", groups: ["server"] },
	{ id: "bun", label: "Bun", groups: ["server"] },
	{ id: "deno", label: "Deno", groups: ["server"] },
	{ id: "python", label: "Python", groups: ["popular", "server"] },
	{ id: "django", label: "Django", groups: ["popular", "server"] },
	{ id: "flask", label: "Flask", groups: ["server"] },
	{ id: "fastapi", label: "FastAPI", groups: ["server"] },
	{ id: "go", label: "Go", groups: ["popular", "server", "desktop"] },
	{ id: "php", label: "PHP", groups: ["popular", "server"] },
	{ id: "laravel", label: "Laravel", groups: ["popular", "server"] },
	{ id: "symfony", label: "Symfony", groups: ["server"] },
	{ id: "ruby", label: "Ruby", groups: ["server"] },
	{ id: "rails", label: "Rails", groups: ["server"] },
	{ id: "java", label: "Java", groups: ["popular", "server"] },
	{ id: "spring", label: "Spring Boot", groups: ["server"] },
	{ id: "dotnet", label: "C# / .NET", groups: ["server", "desktop"] },
	{ id: "elixir", label: "Elixir", groups: ["server"] },
	{ id: "rust", label: "Rust", groups: ["server", "desktop"] },
	{ id: "react-native", label: "React Native", groups: ["popular", "mobile"] },
	{ id: "flutter", label: "Flutter", groups: ["mobile", "desktop"] },
	{ id: "android", label: "Android", groups: ["mobile"] },
	{ id: "ios", label: "iOS", groups: ["mobile"] },
	{ id: "electron", label: "Electron", groups: ["desktop"] },
	{ id: "other", label: "Other", groups: [] },
] as const satisfies readonly { id: string; label: string; groups: readonly PlatformGroup[] }[];

export type PlatformId = (typeof platforms)[number]["id"];
export const platformIds = platforms.map((p) => p.id) as [PlatformId, ...PlatformId[]];
