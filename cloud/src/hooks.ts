import type { Reroute } from "@sveltejs/kit/hooks";

// staff.getblup.io shows the staff pages at its root (staff.getblup.io/contas
// is routes/staff/contas). The address bar keeps the short path. On this
// computer the staff stays at localhost:8787/staff.
export const reroute: Reroute = ({ url }) => {
	if (url.hostname.startsWith("staff.") && !url.pathname.startsWith("/staff")) return `/staff${url.pathname === "/" ? "" : url.pathname}`;
};
