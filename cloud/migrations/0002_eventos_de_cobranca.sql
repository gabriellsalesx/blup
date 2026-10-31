CREATE TABLE `billing_event` (
	`id` integer PRIMARY KEY NOT NULL,
	`stripe_event_id` text NOT NULL,
	`user_id` text,
	`type` text NOT NULL,
	`amount` integer,
	`currency` text,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `billing_event_stripe_event_id_unique` ON `billing_event` (`stripe_event_id`);--> statement-breakpoint
CREATE INDEX `billing_event_user_idx` ON `billing_event` (`user_id`,`id`);