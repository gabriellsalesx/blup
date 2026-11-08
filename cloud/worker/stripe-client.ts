import Stripe from "stripe";

// Matches stripe-node 23.1.0 and the current stable API, not the preview.
// Calls without their own options (the Better Auth plugin's e-mail sync, the
// cron that cancels deleted accounts) wait 15 seconds, not the SDK's 80.
export function createStripe(secret:string){
	return new Stripe(secret,{apiVersion:"2026-10-28.endive",httpClient:Stripe.createFetchHttpClient(),timeout:15_000});
}
