CREATE TABLE `ai_usage` (
	`id` integer PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`kind` text NOT NULL,
	`units` integer NOT NULL,
	`cost_micros` integer NOT NULL,
	`created_at` integer DEFAULT (cast(unixepoch('subsecond') * 1000 as integer)) NOT NULL
);
--> statement-breakpoint
CREATE INDEX `ai_usage_user_idx` ON `ai_usage` (`user_id`,`created_at`);