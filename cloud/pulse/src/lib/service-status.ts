import Circle from "@lucide/svelte/icons/circle";
import CircleMinus from "@lucide/svelte/icons/circle-minus";
import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
import CircleCheck from "@lucide/svelte/icons/circle-check";
import {t} from "#lib/i18n.svelte.ts";
import type {ServiceStatus} from "./status-pages";
// How an incident leaves a service: its mark (a Lucide icon in its color)
// and its name, the same in the form and on the incident's page.
export const serviceIcon={
 none:{Icon:Circle,class:"text-neutral-400"},
 downtime:{Icon:CircleMinus,class:"text-red-500"},
 degraded:{Icon:TriangleAlert,class:"text-amber-500"},
 resolved:{Icon:CircleCheck,class:"text-emerald-500"},
} as const satisfies Record<ServiceStatus,unknown>;
export const serviceName=(s:ServiceStatus)=>s==="none"?t("pulseNotAffected"):s==="downtime"?t("pulseDown"):s==="degraded"?t("pulseDegraded"):t("pulseResolved");
export const serviceStatuses=["none","downtime","degraded","resolved"] as const satisfies readonly ServiceStatus[];
