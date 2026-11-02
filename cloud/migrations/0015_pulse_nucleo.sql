CREATE TABLE `pulse_account` (
	`user_id` text PRIMARY KEY NOT NULL,
	`last_plan` text NOT NULL,
	`read_only_until` integer,
	`purge_started` integer
);
--> statement-breakpoint
CREATE INDEX `pulse_account_purge_idx` ON `pulse_account` (`read_only_until`);--> statement-breakpoint
CREATE TABLE `pulse_grant` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`project_id` text NOT NULL,
	`month` text NOT NULL,
	`amount` integer NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_grant_owner_month_idx` ON `pulse_grant` (`user_id`,`month`);--> statement-breakpoint
CREATE INDEX `pulse_grant_project_idx` ON `pulse_grant` (`project_id`);--> statement-breakpoint
CREATE TABLE `pulse_key` (
	`id` text PRIMARY KEY NOT NULL,
	`project_id` text NOT NULL,
	`hash` text NOT NULL,
	`prefix` text NOT NULL,
	`revoked_at` integer,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `pulse_key_hash_unique` ON `pulse_key` (`hash`);--> statement-breakpoint
CREATE INDEX `pulse_key_project_idx` ON `pulse_key` (`project_id`);--> statement-breakpoint
CREATE TABLE `pulse_month` (
	`user_id` text NOT NULL,
	`month` text NOT NULL,
	`reserved` integer DEFAULT 0 NOT NULL,
	PRIMARY KEY(`user_id`, `month`)
);
--> statement-breakpoint
CREATE TABLE `pulse_project` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`name` text NOT NULL,
	`platform` text NOT NULL,
	`object_name` text NOT NULL,
	`active` integer DEFAULT 1 NOT NULL,
	`deleted_at` integer,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `pulse_project_object_name_unique` ON `pulse_project` (`object_name`);--> statement-breakpoint
CREATE INDEX `pulse_project_owner_idx` ON `pulse_project` (`user_id`,`id`);--> statement-breakpoint
CREATE INDEX `pulse_project_deleted_idx` ON `pulse_project` (`deleted_at`);