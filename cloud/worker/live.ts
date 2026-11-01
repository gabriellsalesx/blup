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
	async fetch(request: Request): Promise<Response> {
		if (request.headers.get("Upgrade") !== "websocket") return new Response("Expected WebSocket", { status: 426 });
		const [client, server] = Object.values(new WebSocketPair());
		this.ctx.acceptWebSocket(server);
		return new Response(null, { status: 101, webSocket: client });
	}

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
	async webSocketMessage(): Promise<void> {}

	async webSocketClose(ws: WebSocket, code: number, reason: string): Promise<void> {
		ws.close(code, reason);
	}
}
