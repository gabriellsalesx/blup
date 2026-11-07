CREATE TABLE `deals_site` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`channel_id` text NOT NULL,
	`slug` text NOT NULL,
	`title` text NOT NULL,
	`description` text DEFAULT '' NOT NULL,
	`language` text DEFAULT 'pt' NOT NULL,
	`logo_light` text,
	`logo_dark` text,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `deals_site_user_id_unique` ON `deals_site` (`user_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `deals_site_channel_id_unique` ON `deals_site` (`channel_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `deals_site_slug_unique` ON `deals_site` (`slug`);--> statement-breakpoint
CREATE TABLE `deals_site_domain` (
	`site_id` text PRIMARY KEY NOT NULL,
	`hostname` text NOT NULL,
	`proof` text NOT NULL,
	`provider_id` text,
	`verified_at` integer,
	`status` text DEFAULT 'pending_ownership' NOT NULL,
	`ssl` text DEFAULT 'pending' NOT NULL,
	`records` text DEFAULT '[]' NOT NULL,
	`checked_at` integer
);
--> statement-breakpoint
CREATE INDEX `deals_site_domain_host_idx` ON `deals_site_domain` (`hostname`,`verified_at`);--> statement-breakpoint
CREATE INDEX `deals_post_channel_sent_idx` ON `deals_post` (`channel_id`,`state`,`sent_at`);