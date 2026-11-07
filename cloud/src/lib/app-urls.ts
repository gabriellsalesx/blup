import { pulseAppURL, dealsAppURL } from "../../shared/panel-paths";
// Where each panel is, for the links between them (the switcher, "see
// your plan"). On this computer (localhost) the panels run side by side:
// the Cloud's Worker (wrangler dev, port 18820, with the staff at /staff)
// and the Pulse and Deals panels (vite, ports 5174 and 5175).
const local = typeof location !== "undefined" && (location.hostname === "localhost" || location.hostname === "127.0.0.1");
export const appURLs = local
	? { cloud: "http://localhost:18820", pulse: "http://localhost:5174/", deals: "http://localhost:5175/" }
	: { cloud: "https://cloud.getblup.io", pulse: pulseAppURL, deals: dealsAppURL };
