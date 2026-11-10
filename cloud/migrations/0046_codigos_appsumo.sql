-- Códigos do AppSumo (plano vitalício, até 5 por conta): só o HMAC do código, nunca o código; e as tentativas erradas por conta na hora.
CREATE TABLE `appsumo_attempt` (
	`user_id` text PRIMARY KEY NOT NULL,
	`window_start` integer NOT NULL,
	`failures` integer NOT NULL
);
--> statement-breakpoint
CREATE TABLE `appsumo_code` (
	`code_hash` text PRIMARY KEY NOT NULL,
	`last4` text NOT NULL,
	`status` text DEFAULT 'available' NOT NULL,
	`user_id` text,
	`imported_at` integer NOT NULL,
	`redeemed_at` integer,
	`revoked_at` integer,
	`revoke_reason` text
);
--> statement-breakpoint
CREATE INDEX `appsumo_code_user_idx` ON `appsumo_code` (`user_id`,`status`);