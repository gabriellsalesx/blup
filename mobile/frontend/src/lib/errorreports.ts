import type { ErrorReports } from "../../bindings/github.com/gabriellsalesx/blup/mobile/models";

// Os erros da tela do app, para o Blup Pulse, só com o "sim" da pessoa
// (errors.go guarda a resposta e entrega a DSN). Só erros: sem rastreio,
// sem replay, sem linhas do console, nada das notas; a pasta do app e a
// pessoal viram "~".
// https://docs.sentry.io/platforms/javascript/guides/svelte/
let started = false;
// The SDK loads only after the "yes": without it, the app never pays for it.
const sdk = () => import("./sentry");
const privateDirs = /(?:\/data\/(?:user\/\d+|data)\/[^/\s]+|\/home\/[^/\s]+|\/Users\/[^/\s]+)/g;
const scrub = (s?: string) => s?.replace(privateDirs, "~");

export function startPageReports(r: ErrorReports | null) {
  if (!r?.dsn) {
    if (started) void sdk().then((Sentry) => Sentry.close());
    started = false;
    return;
  }
  if (started) return;
  started = true;
  void sdk().then((Sentry) => started && Sentry.init({
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
      // As in errors.go: the trail keeps what happened, never its data. Taps
      // and typing (ui.*) go out entirely: a tapped element's title can be a
      // note's name.
      event.breadcrumbs = event.breadcrumbs?.filter((b) => !b.category?.startsWith("ui."));
      for (const b of event.breadcrumbs ?? []) { b.message = scrub(b.message); delete b.data; }
      return event;
    },
  })).catch(() => { started = false; });
}
