// Asking before anything that can't be undone: `await confirm({...})`
// resolves true only if the person confirms. One dialog serves the whole
// app (ConfirmHost in the root layout). `typeToConfirm` asks to type a word
// (or the e-mail) first, for the gravest actions.

export type ConfirmOptions = {
	title: string;
	message: string;
	confirmLabel?: string;
	danger?: boolean;
	typeToConfirm?: string;
};

// `typed` is what the person typed in the confirmation field.
export const confirmState = $state<{ open: boolean; opts: ConfirmOptions | null; typed: string }>({ open: false, opts: null, typed: "" });
let resolver: ((ok: boolean) => void) | null = null;

export function confirm(opts: ConfirmOptions): Promise<boolean> {
	resolver?.(false);
	return new Promise((resolve) => {
		resolver = resolve;
		confirmState.opts = opts;
		confirmState.typed = "";
		confirmState.open = true;
	});
}

export function answer(ok: boolean) {
	confirmState.open = false;
	resolver?.(ok);
	resolver = null;
}
