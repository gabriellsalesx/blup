// The front door of cloud.getblup.io and staff.getblup.io, at the edge
// (wrangler.edge.jsonc). The Cloud's Worker runs near the database (Smart
// Placement), which is right for the API but made every page of the panel
// travel there and back too. Here the pages and their files come from the
// nearest location without running any code (assets first); only what
// needs the database (run_worker_first: /api, /staff, /_blup) gets to this
// Worker, which hands the request, untouched, to the Cloud's Worker.

interface Env {
	BACKEND: Fetcher;
}

export default {
	fetch(req: Request, env: Env): Promise<Response> {
		return env.BACKEND.fetch(req);
	},
} satisfies ExportedHandler<Env>;
