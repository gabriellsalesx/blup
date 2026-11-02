CREATE TABLE `pulse_channel` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`type` text NOT NULL,
	`target` text NOT NULL,
	`label` text NOT NULL,
	`active` integer DEFAULT 1 NOT NULL,
	`last_error` text,
	`next_at` integer DEFAULT 0 NOT NULL,
	`test_at` integer DEFAULT 0 NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_channel_owner_idx` ON `pulse_channel` (`user_id`);--> statement-breakpoint
CREATE TABLE `pulse_cooldown` (
	`key` text PRIMARY KEY NOT NULL,
	`until` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_cooldown_expiry_idx` ON `pulse_cooldown` (`until`);--> statement-breakpoint
CREATE TABLE `pulse_delivery_budget` (
	`key` text PRIMARY KEY NOT NULL,
	`next_at` integer NOT NULL
);
--> statement-breakpoint
CREATE TABLE `pulse_outbox` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`project_id` text,
	`channel_id` text NOT NULL,
	`ref_id` text NOT NULL,
	`kind` text NOT NULL,
	`created_at` integer NOT NULL,
	`expires_at` integer NOT NULL,
	`next_at` integer NOT NULL,
	`attempts` integer DEFAULT 0 NOT NULL,
	`state` text DEFAULT 'pending' NOT NULL,
	`claim` text,
	`claim_until` integer DEFAULT 0 NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_outbox_due_idx` ON `pulse_outbox` (`state`,`next_at`);--> statement-breakpoint
CREATE INDEX `pulse_outbox_expiry_idx` ON `pulse_outbox` (`expires_at`);--> statement-breakpoint
CREATE INDEX `pulse_outbox_channel_idx` ON `pulse_outbox` (`channel_id`);--> statement-breakpoint
CREATE INDEX `pulse_outbox_owner_idx` ON `pulse_outbox` (`user_id`);--> statement-breakpoint
CREATE TABLE `pulse_project_channel` (
	`project_id` text NOT NULL,
	`channel_id` text NOT NULL,
	`type` text NOT NULL,
	PRIMARY KEY(`project_id`, `type`)
);
--> statement-breakpoint
CREATE INDEX `pulse_project_channel_target_idx` ON `pulse_project_channel` (`channel_id`);--> statement-breakpoint
CREATE TABLE `pulse_telegram_link` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`hash` text NOT NULL,
	`expires_at` integer NOT NULL,
	`state` text DEFAULT 'pending' NOT NULL,
	`chat_id` text,
	`chat_label` text,
	`channel_id` text
);
--> statement-breakpoint
CREATE UNIQUE INDEX `pulse_telegram_link_hash_unique` ON `pulse_telegram_link` (`hash`);--> statement-breakpoint
CREATE INDEX `pulse_telegram_link_owner_idx` ON `pulse_telegram_link` (`user_id`);--> statement-breakpoint
CREATE INDEX `pulse_telegram_link_expiry_idx` ON `pulse_telegram_link` (`expires_at`);--> statement-breakpoint
CREATE TABLE `pulse_telegram_update` (
	`id` integer PRIMARY KEY NOT NULL,
	`received_at` integer NOT NULL,
	`applied` integer DEFAULT 0 NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_telegram_update_expiry_idx` ON `pulse_telegram_update` (`received_at`);--> statement-breakpoint
ALTER TABLE `pulse_project` ADD `alert_checked_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `pulse_project_alert_idx` ON `pulse_project` (`alert_checked_at`);--> statement-breakpoint
CREATE INDEX `pulse_grant_month_idx` ON `pulse_grant` (`month`);