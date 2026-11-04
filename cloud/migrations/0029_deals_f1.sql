CREATE TABLE `deals_telegram_link` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`hash` text NOT NULL,
	`state` text DEFAULT 'pending' NOT NULL,
	`expires_at` integer NOT NULL,
	`telegram_user` text,
	`chat_id` text,
	`chat_label` text,
	`channel_id` text
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_telegram_link_hash_unique` ON `deals_telegram_link` (`hash`);--> statement-breakpoint
CREATE INDEX `deals_telegram_link_owner_idx` ON `deals_telegram_link` (`user_id`);--> statement-breakpoint
CREATE INDEX `deals_telegram_link_user_idx` ON `deals_telegram_link` (`telegram_user`,`state`);--> statement-breakpoint
CREATE TABLE `deals_telegram_update` (
	`id` integer PRIMARY KEY NOT NULL,
	`received_at` integer NOT NULL,
	`applied` integer DEFAULT 0 NOT NULL
);
--> statement-breakpoint
CREATE INDEX `deals_telegram_update_received_idx` ON `deals_telegram_update` (`received_at`);--> statement-breakpoint
ALTER TABLE `deals_affiliate` ADD `hint` text;--> statement-breakpoint
ALTER TABLE `deals_channel` ADD `target_hash` text;--> statement-breakpoint
ALTER TABLE `deals_channel` ADD `next_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `deals_channel` ADD `last_error` text;--> statement-breakpoint
CREATE INDEX `deals_channel_hash_idx` ON `deals_channel` (`target_hash`);--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `kind` text DEFAULT 'product' NOT NULL;--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `platforms` text DEFAULT '[]' NOT NULL;--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `discount` integer;--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `credit` text;--> statement-breakpoint
CREATE INDEX `deals_offer_kind_idx` ON `deals_offer` (`kind`,`first_seen_at`);--> statement-breakpoint
ALTER TABLE `deals_post` ADD `affiliate` integer DEFAULT 0 NOT NULL;