// What a session's User-Agent says about where it is: the system (its
// mark in the panels) and a short name. Blup Studio names itself
// ("Blup Studio (Linux; my-pc)"; older ones sent Go's names, "android");
// a browser is read by its usual words ("Chrome on Linux").

export const devicePlatforms = ["android", "ios", "macos", "linux", "windows", "other"] as const;
export type DevicePlatform = (typeof devicePlatforms)[number];

const systemNames = { android: "Android", ios: "iOS", macos: "macOS", linux: "Linux", windows: "Windows" } as const satisfies Record<Exclude<DevicePlatform, "other">, string>;
const studioSystems: Partial<Record<string, Exclude<DevicePlatform, "other">>> = { linux: "linux", macos: "macos", darwin: "macos", windows: "windows", android: "android", ios: "ios" };

export type AgentInfo = {
	platform: DevicePlatform;
	// The system's name ("Linux"), or null when it is not known.
	system: string | null;
	// Blup Studio, with the computer's own name.
	studio: { host: string } | null;
	// A browser's name ("Chrome"), or null.
	browser: string | null;
};

export function readAgent(agent: string | null | undefined): AgentInfo {
	const text = agent ?? "";
	const studio = /^Blup Studio \((.+?);\s*(.+)\)/.exec(text);
	if (studio) {
		const known = studioSystems[studio[1]!.trim().toLowerCase()];
		return { platform: known ?? "other", system: known ? systemNames[known] : studio[1]!.trim(), studio: { host: studio[2]!.trim() }, browser: null };
	}
	// iPhones and iPads before Macs ("like Mac OS X"), Android before Linux.
	const platform: DevicePlatform = /Android/i.test(text) ? "android" : /iPhone|iPad|iPod/i.test(text) ? "ios" : /Windows/i.test(text) ? "windows" : /Macintosh|Mac OS X/i.test(text) ? "macos" : /Linux|X11|CrOS/i.test(text) ? "linux" : "other";
	// Edge and Opera say Chrome too; Chrome says Safari too.
	const browser = /Edg(?:e|A|iOS)?\//.test(text) ? "Edge" : /OPR\/|Opera/.test(text) ? "Opera" : /Firefox\/|FxiOS\//.test(text) ? "Firefox" : /Chrome\/|CriOS\//.test(text) ? "Chrome" : /Safari\//.test(text) ? "Safari" : null;
	return { platform, system: platform === "other" ? null : systemNames[platform], studio: null, browser };
}
