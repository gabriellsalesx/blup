// First installation only. Creates the Cloud DO namespaces before Sites
// can reference them. No public routes, assets, cron or external services.
export {SyncLive} from "./live";
export {PulseProject} from "./pulse/project";
export default {
 fetch(){return new Response("Service unavailable",{status:503,headers:{"cache-control":"no-store"}});},
} satisfies ExportedHandler<Cloudflare.Env>;
