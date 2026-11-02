// What an account uses (the API's `usage`) and how to show it.
import {intlLocale} from "./i18n.svelte";
import type { Me } from "./api";
export type Usage = Me["usage"];

export const bytes = (n: number) => {
	if (n === 0) return "0 MB";
	if (n < 1024 * 1024) return `${Math.round(n / 1024).toLocaleString(intlLocale())} KB`;
	if (n < 1024 ** 3) return `${(n / 1024 ** 2).toLocaleString(intlLocale(), { maximumFractionDigits: 1 })} MB`;
	return `${(n / 1024 ** 3).toLocaleString(intlLocale(), { maximumFractionDigits: 1 })} GB`;
};

// A plan's storage limit is in MB.
export const megabytes = (mb: number) => (mb >= 1024 ? `${(mb / 1024).toLocaleString(intlLocale())} GB` : `${mb.toLocaleString(intlLocale())} MB`);
