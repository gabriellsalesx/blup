CREATE TABLE `stripe_webhook_event` (
	`id` text PRIMARY KEY NOT NULL,
	`type` text NOT NULL,
	`user_id` text,
	`processed_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `stripe_webhook_event_processed_idx` ON `stripe_webhook_event` (`processed_at`);--> statement-breakpoint
CREATE INDEX `stripe_webhook_event_user_idx` ON `stripe_webhook_event` (`user_id`);--> statement-breakpoint
CREATE INDEX `subscription_stripe_id_idx` ON `subscription` (`stripe_subscription_id`);