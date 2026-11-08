-- Nonces dos avisos da Ponte (WhatsApp) vistos nos últimos minutos: um aviso repetido é recusado.
CREATE TABLE `deals_bridge_nonce` (
	`nonce` text PRIMARY KEY NOT NULL,
	`received_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `deals_bridge_nonce_received_idx` ON `deals_bridge_nonce` (`received_at`);