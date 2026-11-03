CREATE TABLE `pulse_sourcemap` (
	`project_id` text NOT NULL,
	`release` text NOT NULL,
	`file` text NOT NULL,
	`object_key` text NOT NULL,
	`size` integer NOT NULL,
	`uploaded_at` integer NOT NULL,
	PRIMARY KEY(`project_id`, `release`, `file`)
);
--> statement-breakpoint
CREATE INDEX `pulse_sourcemap_uploaded_idx` ON `pulse_sourcemap` (`uploaded_at`);--> statement-breakpoint
CREATE TABLE `pulse_upload_token` (
	`id` text PRIMARY KEY NOT NULL,
	`project_id` text NOT NULL,
	`hash` text NOT NULL,
	`prefix` text NOT NULL,
	`created_at` integer NOT NULL,
	`revoked_at` integer
);
--> statement-breakpoint
CREATE UNIQUE INDEX `pulse_upload_token_hash_unique` ON `pulse_upload_token` (`hash`);--> statement-breakpoint
CREATE INDEX `pulse_upload_token_project_idx` ON `pulse_upload_token` (`project_id`);