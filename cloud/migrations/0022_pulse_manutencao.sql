CREATE TABLE `pulse_maintenance` (
	`id` text PRIMARY KEY NOT NULL,
	`page_id` text NOT NULL,
	`title` text NOT NULL,
	`body` text NOT NULL,
	`starts_at` integer NOT NULL,
	`ends_at` integer NOT NULL,
	`created_at` integer NOT NULL,
	`updated_at` integer NOT NULL,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `pulse_maintenance_page_idx` ON `pulse_maintenance` (`page_id`,`starts_at`);--> statement-breakpoint
CREATE INDEX `pulse_maintenance_expiry_idx` ON `pulse_maintenance` (`expires_at`);--> statement-breakpoint
CREATE TABLE `pulse_maintenance_component` (
	`maintenance_id` text NOT NULL,
	`monitor_id` text NOT NULL,
	PRIMARY KEY(`maintenance_id`, `monitor_id`)
);
--> statement-breakpoint
CREATE INDEX `pulse_maintenance_component_monitor_idx` ON `pulse_maintenance_component` (`monitor_id`);