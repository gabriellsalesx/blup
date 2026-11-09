// The panels' texts, one file per language (/i18n/<lang>.json in the build),
// made from src/lib/i18n/catalog.ts. The catalog carries every key in the
// three languages and the texts of the three panels: in the JavaScript, a
// page would download all of it to show one language. The file of the
// language in use starts downloading in the <head> (static/theme.js) and
// the panel waits for it before its first page (src/lib/i18n.svelte.ts).
// The catalog stays the source and the types; in `vite dev` there are no
// files and the panel reads the catalog itself.
import { runnerImport, type Plugin } from "vite";

const languages = ["pt", "en", "es"] as const;

/** catalog: the path of src/lib/i18n/catalog.ts from the panel's root. */
export function i18nSplit(catalog: string): Plugin {
	let root = "";
	return {
		name: "blup-i18n-split",
		apply: "build",
		configResolved(config) {
			root = config.root;
		},
		async generateBundle() {
			if (this.environment.name !== "client") return;
			const { module } = await runnerImport<{ messages: Record<string, readonly [string, string, string]> }>(`${root}/${catalog}`, { configFile: false });
			languages.forEach((lang, i) => {
				const table = Object.fromEntries(Object.entries(module.messages).map(([key, texts]) => [key, texts[i]]));
				this.emitFile({ type: "asset", fileName: `i18n/${lang}.json`, source: JSON.stringify(table) });
			});
		},
	};
}
