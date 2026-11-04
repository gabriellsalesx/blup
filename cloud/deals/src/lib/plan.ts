import {panel} from "./panel.svelte";
// Changes (a new hunt, a channel, a post) show only with the Deals plan on
// and the panel not read-only.
export const hasPlan=()=>!!panel.usage?.plan&&!panel.usage.readOnly;
