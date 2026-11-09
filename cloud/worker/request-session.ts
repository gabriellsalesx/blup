import { auth, type Session } from "./auth";

// The session of one request, looked up once: the rate limit and the
// route's own guard both need it, and each lookup is a trip to D1. Keyed
// by the Request object, so nothing is shared between requests (or kept
// after one ends).
const looked = new WeakMap<Request, Promise<Session | null>>();
export function requestSession(req: Request): Promise<Session | null> {
	let pending = looked.get(req);
	if (!pending) {
		pending = auth.api.getSession({ headers: req.headers });
		looked.set(req, pending);
		// A failed lookup is not remembered: the next caller asks again.
		pending.catch(() => looked.delete(req));
	}
	return pending;
}
