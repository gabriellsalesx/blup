import Stripe from "stripe";

// Matches stripe-node 23.1.0 and the current stable API, not the preview.
export function createStripe(secret:string){
	return new Stripe(secret,{apiVersion:"2026-10-28.endive",httpClient:Stripe.createFetchHttpClient()});
}
