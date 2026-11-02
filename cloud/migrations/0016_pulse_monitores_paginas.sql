CREATE TABLE `pulse_component` (
	`page_id` text NOT NULL,
	`monitor_id` text NOT NULL,
	`name` text NOT NULL,
	`position` integer NOT NULL,
	PRIMARY KEY(`page_id`, `monitor_id`)
);
--> statement-breakpoint
CREATE TABLE `pulse_incident` (
	`id` text PRIMARY KEY NOT NULL,
	`page_id` text NOT NULL,
	`title` text NOT NULL,
	`state` text NOT NULL,
	`impact` text NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_incident_page_idx` ON `pulse_incident` (`page_id`,`created_at`);--> statement-breakpoint
CREATE TABLE `pulse_incident_update` (
	`id` text PRIMARY KEY NOT NULL,
	`incident_id` text NOT NULL,
	`body` text NOT NULL,
	`state` text NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_incident_update_idx` ON `pulse_incident_update` (`incident_id`,`created_at`);--> statement-breakpoint
CREATE TABLE `pulse_monitor` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`project_id` text NOT NULL,
	`name` text NOT NULL,
	`url` text NOT NULL,
	`interval_minutes` integer NOT NULL,
	`active` integer DEFAULT 1 NOT NULL,
	`next_at` integer NOT NULL,
	`claim` text,
	`claim_until` integer DEFAULT 0 NOT NULL,
	`state` text DEFAULT 'unknown' NOT NULL,
	`revision` integer DEFAULT 1 NOT NULL,
	`last_at` integer,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_monitor_owner_idx` ON `pulse_monitor` (`user_id`,`id`);--> statement-breakpoint
CREATE INDEX `pulse_monitor_due_idx` ON `pulse_monitor` (`active`,`next_at`);--> statement-breakpoint
CREATE INDEX `pulse_monitor_project_idx` ON `pulse_monitor` (`project_id`);--> statement-breakpoint
CREATE TABLE `pulse_page` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`slug` text NOT NULL,
	`title` text NOT NULL,
	`published` integer DEFAULT 0 NOT NULL,
	`language` text DEFAULT 'en' NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `pulse_page_slug_unique` ON `pulse_page` (`slug`);--> statement-breakpoint
CREATE INDEX `pulse_page_owner_idx` ON `pulse_page` (`user_id`,`id`);--> statement-breakpoint
CREATE TABLE `pulse_page_domain` (
	`page_id` text PRIMARY KEY NOT NULL,
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
CREATE INDEX `pulse_domain_host_idx` ON `pulse_page_domain` (`hostname`,`verified_at`);--> statement-breakpoint
ALTER TABLE `pulse_project` ADD `cleaned_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `pulse_project_cleaned_idx` ON `pulse_project` (`cleaned_at`);