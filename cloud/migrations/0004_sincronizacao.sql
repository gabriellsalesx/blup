CREATE TABLE `sync_account` (
	`user_id` text PRIMARY KEY NOT NULL,
	`wrapped_key` text NOT NULL,
	`previous_key` text,
	`storage_bytes` integer DEFAULT 0 NOT NULL,
	`last_sync_at` integer,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL,
	`updated_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL
);
--> statement-breakpoint
CREATE TABLE `sync_block` (
	`user_id` text NOT NULL,
	`id` text NOT NULL,
	`size` integer NOT NULL,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL,
	PRIMARY KEY(`user_id`, `id`)
);
--> statement-breakpoint
CREATE TABLE `vault` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`name` text NOT NULL,
	`head` integer DEFAULT 0 NOT NULL,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL,
	`updated_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL
);
--> statement-breakpoint
CREATE INDEX `vault_user_idx` ON `vault` (`user_id`,`id`);--> statement-breakpoint
CREATE TABLE `vault_version` (
	`vault_id` text NOT NULL,
	`n` integer NOT NULL,
	`key` text NOT NULL,
	`size` integer NOT NULL,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL,
	PRIMARY KEY(`vault_id`, `n`)
);
