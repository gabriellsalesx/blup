-- O aparelho de cada sessão do Blup Studio e do Blup Notes (HMAC do aparelho, por conta): o mesmo aparelho entrando de novo troca a sessão antiga em vez de contar outra vez; e o aparelho de cada código de entrada até ele ser usado.
CREATE TABLE `device_pending` (
	`code_hash` text PRIMARY KEY NOT NULL,
	`device` text NOT NULL,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `device_pending_expiry_idx` ON `device_pending` (`expires_at`);--> statement-breakpoint
ALTER TABLE `session` ADD `device_id` text;--> statement-breakpoint
CREATE INDEX `session_user_device_idx` ON `session` (`user_id`,`device_id`);