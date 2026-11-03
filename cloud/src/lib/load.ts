import { errorMessage } from "#lib/i18n/errors.ts";
import { toast } from "#lib/toast.svelte.ts";

// A page's request in its `load`: it starts at once, alongside the layout's
// checks (`parent`), and is read only after them, so a signed-out visit
// still just goes to sign-in. A failure is a toast and `fallback`, and the
// page opens without it, as before (`message: false` keeps it quiet).
export async function loaded<T, F = T>(request: Promise<T>, parent: () => Promise<unknown>, fallback: F, message?: string | false): Promise<T | F> {
	const settled = request.then((value) => ({ ok: true as const, value }), (error: unknown) => ({ ok: false as const, error }));
	await parent();
	const result = await settled;
	if (result.ok) return result.value;
	if (message !== false) toast(message ?? (() => errorMessage(result.error)), { kind: "error" });
	return fallback;
}
