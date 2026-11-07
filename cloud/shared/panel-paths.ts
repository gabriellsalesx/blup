// Addresses of the Blup Cloud panel used from outside its own SvelteKit app
// (the Pulse panel, the Worker's e-mails and Stripe returns). Inside an
// app, links go through `resolve()` from `$app/paths`, which checks them
// against the routes at build time. Paths and query names are English; the
// pages themselves are translated.
export const cloudPaths = {
	signIn: "/sign-in",
	account: "/account",
	plan: "/account/plan",
	sync: "/account/sync",
	sites: "/account/sites",
	pulse: "/account/pulse",
	device: "/device",
} as const satisfies Record<string, `/${string}`>;
export type CloudPath = (typeof cloudPaths)[keyof typeof cloudPaths];

// After signing in, the page that asked for it: /sign-in?next=/account/plan
export const nextParam = "next";
// Stripe's return after a paid checkout: /account/plan?checkout=success
export const checkoutParam = "checkout";

// /sign-in?next=<path>, for a path of the same panel.
export const signInFor = (next: string) => `${cloudPaths.signIn}?${new URLSearchParams({ [nextParam]: next })}`;

// The Pulse panel, opened from the Cloud panel's switcher and after signing in.
export const pulseAppURL = "https://pulse.getblup.io/";
// The Deals panel, opened from the other panels' switcher.
export const dealsAppURL = "https://deals.getblup.io/";
