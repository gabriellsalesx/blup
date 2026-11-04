import {createBlupAuthClient} from "#lib/auth.ts";
import {cloudURL} from "./api";
// The Blup sign-in from the Deals panel: the Cloud's Worker, with its
// cookies, so one sign-in opens every panel.
export const dealsAuth=createBlupAuthClient(cloudURL);
