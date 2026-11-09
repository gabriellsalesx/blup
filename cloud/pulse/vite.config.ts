import adapter from "@sveltejs/adapter-static";
import {sveltekit} from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import {defineConfig} from "vite";
import {sentryConnect,statsConnect,statsScript} from "../shared/sentry-csp.ts";
import {i18nSplit} from "../shared/i18n-split.ts";
// The CSP: only our files run, plus Cloudflare Turnstile on the sign-in
// (the same as the Cloud panel's); the data and the sign-in come from the
// Cloud's Worker; the errors go to Blup Pulse's ingest.
export default defineConfig({
 plugins:[tailwindcss(),i18nSplit("../src/lib/i18n/catalog.ts"),sveltekit({adapter:adapter({fallback:"index.html"}),csp:{mode:"hash",directives:{"default-src":["self"],"script-src":["self","https://challenges.cloudflare.com", ...statsScript],"frame-src":["https://challenges.cloudflare.com","https://pulse.getblup.site"],"connect-src":["self","https://cloud.getblup.io","https://challenges.cloudflare.com",...sentryConnect(), ...statsConnect],"img-src":["self","data:","blob:"],"style-src":["self","unsafe-inline"],"font-src":["self"],"object-src":["none"],"base-uri":["self"],"form-action":["self"]}}})],
 server:{fs:{allow:[".."]}},
});
