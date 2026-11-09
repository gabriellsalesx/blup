import {closeCodeToSend} from "./live-close";
import { DurableObject } from "cloudflare:workers";

// The live room of an account (plans with live sync): every Blup Studio
// signed in keeps a WebSocket here, and when one saves a new version of a
// vault the others hear "vault X is at version N" at once. With the
// Hibernation API the room sleeps between messages, so an idle connection
// costs nothing. Only ids and numbers pass through (never notes).
export class SyncLive extends DurableObject<Cloudflare.Env> {
	constructor(ctx: DurableObjectState, env: Cloudflare.Env) {
		super(ctx, env);
		// Keep-alive pings answered without waking the room.
		this.ctx.setWebSocketAutoResponse(new WebSocketRequestResponsePair("ping", "pong"));
	}

	// A Studio joining (the Worker already checked the session and plan).
	override async fetch(request: Request): Promise<Response> {
		if (request.headers.get("Upgrade") !== "websocket") return new Response("Expected WebSocket", { status: 426 });
		const { 0: client, 1: server } = new WebSocketPair();
		this.ctx.acceptWebSocket(server);
		return new Response(null, { status: 101, webSocket: client });
	}

	async revoke():Promise<void>{for(const ws of this.ctx.getWebSockets()){try{ws.close(1008,"Session ended");}catch{}}}

	// Called by the Worker after a new version is saved.
	async notify(vault: string, n: number): Promise<void> {
		const message = JSON.stringify({ vault, n });
		for (const ws of this.ctx.getWebSockets()) {
			try {
				ws.send(message);
			} catch {
				// a socket closing meanwhile
			}
		}
	}

	// Studios only listen; anything they send is ignored.
	override async webSocketMessage(): Promise<void> {}

	// The close is answered with the same code, except the ones only a
	// receiver may see (1005 none sent, 1006 dropped without a close, 1015
	// TLS): sending those throws, so a dropped connection says 1000.
	override async webSocketClose(ws: WebSocket, code: number, reason: string): Promise<void> {
		try{ws.close(closeCodeToSend(code), reason);}catch{/* already closed */}
	}
}
