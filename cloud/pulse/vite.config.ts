import adapter from "@sveltejs/adapter-static";
import {sveltekit} from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import {defineConfig} from "vite";
export default defineConfig({
 plugins:[tailwindcss(),sveltekit({adapter:adapter({fallback:"index.html"}),csp:{mode:"hash",directives:{"default-src":["self"],"script-src":["self"],"connect-src":["self","https://cloud.getblup.io"],"img-src":["self","data:"],"style-src":["self","unsafe-inline"],"font-src":["self"],"object-src":["none"],"base-uri":["self"],"form-action":["self"]}}})],
 server:{fs:{allow:[".."]}},
});
