import {createBlupAuthClient} from "#lib/auth.ts";
import {cloudURL} from "./api";
// The Blup sign-in from the Pulse panel: the Cloud's Worker, with its
// cookies, so one sign-in opens both panels.
export const pulseAuth=createBlupAuthClient(cloudURL);
