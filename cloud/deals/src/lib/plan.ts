import {panel} from "./panel.svelte";
// Actions (new hunt, connect channel...) show only with the Deals plan on.
export const hasPlan=()=>!!panel.usage?.plan;
