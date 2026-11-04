CREATE TABLE `deals_affiliate` (
	`user_id` text NOT NULL,
	`store` text NOT NULL,
	`tag` text,
	`credentials` text,
	`level` text,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL,
	PRIMARY KEY(`user_id`, `store`)
);
--> statement-breakpoint
CREATE TABLE `deals_channel` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`platform` text NOT NULL,
	`name` text NOT NULL,
	`target` text NOT NULL,
	`guild_id` text,
	`active` integer DEFAULT 1 NOT NULL,
	`verified_at` integer,
	`created_at` integer NOT NULL,
	`plan_disabled` integer DEFAULT 0 NOT NULL,
	`retire_at` integer
);
--> statement-breakpoint
CREATE INDEX `deals_channel_owner_idx` ON `deals_channel` (`user_id`,`id`);--> statement-breakpoint
CREATE UNIQUE INDEX `deals_channel_target_idx` ON `deals_channel` (`user_id`,`platform`,`target`);--> statement-breakpoint
CREATE INDEX `deals_channel_retire_idx` ON `deals_channel` (`retire_at`);--> statement-breakpoint
CREATE TABLE `deals_click_day` (
	`post_id` text NOT NULL,
	`user_id` text NOT NULL,
	`day` text NOT NULL,
	`clicks` integer DEFAULT 0 NOT NULL,
	PRIMARY KEY(`post_id`, `day`)
);
--> statement-breakpoint
CREATE INDEX `deals_click_owner_day_idx` ON `deals_click_day` (`user_id`,`day`);--> statement-breakpoint
CREATE TABLE `deals_hunt` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`name` text NOT NULL,
	`kind` text NOT NULL,
	`source_id` text,
	`filters` text DEFAULT '{}' NOT NULL,
	`channels` text DEFAULT '[]' NOT NULL,
	`interval_minutes` integer DEFAULT 30 NOT NULL,
	`active` integer DEFAULT 1 NOT NULL,
	`last_post_at` integer,
	`created_at` integer NOT NULL,
	`plan_disabled` integer DEFAULT 0 NOT NULL,
	`retire_at` integer
);
--> statement-breakpoint
CREATE INDEX `deals_hunt_owner_idx` ON `deals_hunt` (`user_id`,`id`);--> statement-breakpoint
CREATE INDEX `deals_hunt_source_idx` ON `deals_hunt` (`source_id`);--> statement-breakpoint
CREATE INDEX `deals_hunt_retire_idx` ON `deals_hunt` (`retire_at`);--> statement-breakpoint
CREATE TABLE `deals_offer` (
	`id` text PRIMARY KEY NOT NULL,
	`hash` text NOT NULL,
	`store` text NOT NULL,
	`external_id` text,
	`source_id` text,
	`title` text NOT NULL,
	`url` text NOT NULL,
	`image_url` text,
	`price` integer,
	`original_price` integer,
	`currency` text,
	`first_seen_at` integer NOT NULL,
	`last_seen_at` integer NOT NULL,
	`expires_at` integer
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_offer_hash_unique` ON `deals_offer` (`hash`);--> statement-breakpoint
CREATE INDEX `deals_offer_store_idx` ON `deals_offer` (`store`,`external_id`);--> statement-breakpoint
CREATE INDEX `deals_offer_seen_idx` ON `deals_offer` (`last_seen_at`);--> statement-breakpoint
CREATE INDEX `deals_offer_source_idx` ON `deals_offer` (`source_id`,`last_seen_at`);--> statement-breakpoint
CREATE TABLE `deals_post` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`hunt_id` text,
	`channel_id` text NOT NULL,
	`offer_id` text NOT NULL,
	`code` text NOT NULL,
	`link` text NOT NULL,
	`state` text DEFAULT 'queued' NOT NULL,
	`attempts` integer DEFAULT 0 NOT NULL,
	`next_at` integer NOT NULL,
	`claim` text,
	`claim_until` integer DEFAULT 0 NOT NULL,
	`day` text NOT NULL,
	`sent_at` integer,
	`error` text,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_post_code_unique` ON `deals_post` (`code`);--> statement-breakpoint
CREATE UNIQUE INDEX `deals_post_once_idx` ON `deals_post` (`channel_id`,`offer_id`);--> statement-breakpoint
CREATE INDEX `deals_post_due_idx` ON `deals_post` (`state`,`next_at`);--> statement-breakpoint
CREATE INDEX `deals_post_owner_day_idx` ON `deals_post` (`user_id`,`day`);--> statement-breakpoint
CREATE INDEX `deals_post_created_idx` ON `deals_post` (`created_at`);--> statement-breakpoint
CREATE TABLE `deals_price` (
	`offer_id` text NOT NULL,
	`at` integer NOT NULL,
	`price` integer NOT NULL,
	`currency` text NOT NULL,
	PRIMARY KEY(`offer_id`, `at`)
);
--> statement-breakpoint
CREATE INDEX `deals_price_at_idx` ON `deals_price` (`at`);--> statement-breakpoint
CREATE TABLE `deals_source` (
	`id` text PRIMARY KEY NOT NULL,
	`kind` text NOT NULL,
	`key` text NOT NULL,
	`config` text DEFAULT '{}' NOT NULL,
	`cursor` text,
	`next_at` integer NOT NULL,
	`last_at` integer,
	`failures` integer DEFAULT 0 NOT NULL,
	`error` text,
	`claim` text,
	`claim_until` integer DEFAULT 0 NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_source_key_unique` ON `deals_source` (`key`);--> statement-breakpoint
CREATE INDEX `deals_source_due_idx` ON `deals_source` (`next_at`);