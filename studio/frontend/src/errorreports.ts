import * as Sentry from "@sentry/svelte";

// The page's own errors (the Studio's screens), sent to Blup Pulse only
// when the person said yes (errors.go keeps the answer and gives the DSN).
// Errors only: no tracing, no replay, no console lines, nothing of the
// notes; the home folder (it has the person's name) becomes "~".
// https://docs.sentry.io/platforms/javascript/guides/svelte/
export type Reports = { answer: string; dsn: string; environment: string; release: string };

let started = false;
const home = /(?:\/home\/|\/Users\/|[A-Za-z]:\\Users\\)[^/\\]+/g;
const scrub = (s?: string) => s?.replace(home, "~");

export function startPageReports(r: Reports | null) {
  if (!r?.dsn) {
    if (started) void Sentry.close();
    started = false;
    return;
  }
  if (started) return;
  started = true;
  Sentry.init({
    dsn: r.dsn,
    environment: r.environment,
    release: r.release,
    dataCollection: { userInfo: false, cookies: false, httpHeaders: false, httpBodies: [], urlQueryParams: false },
    sendClientReports: false,
    integrations: (defaults) => defaults.filter((i) => !/console/i.test(i.name)),
    beforeSend(event) {
      delete event.user;
      delete event.request;
      if (event.message) event.message = scrub(event.message);
      for (const ex of event.exception?.values ?? []) {
        ex.value = scrub(ex.value);
        for (const f of ex.stacktrace?.frames ?? []) { f.abs_path = scrub(f.abs_path); f.filename = scrub(f.filename); }
      }
      // As errors.go does: a click's breadcrumb names the element (its
      // title may be a note's name) and a request's carries its address.
      for (const b of event.breadcrumbs ?? []) { b.message = scrub(b.message); delete b.data; }
      event.breadcrumbs = event.breadcrumbs?.filter((b) => !b.category?.startsWith("ui."));
      return event;
    },
  });
}
