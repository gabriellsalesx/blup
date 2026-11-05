CREATE TABLE `deals_whatsapp` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`state` text DEFAULT 'pairing' NOT NULL,
	`phone_end` text,
	`risk_accepted_at` integer NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_whatsapp_user_idx` ON `deals_whatsapp` (`user_id`);--> statement-breakpoint
ALTER TABLE `deals_channel` ADD `session_id` text;