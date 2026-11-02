CREATE TABLE `reminder` (
	`user_id` text NOT NULL,
	`id` text NOT NULL,
	`send_at` integer NOT NULL,
	`sent_at` integer,
	PRIMARY KEY(`user_id`, `id`)
);
--> statement-breakpoint
CREATE INDEX `reminder_due_idx` ON `reminder` (`sent_at`,`send_at`);--> statement-breakpoint
CREATE TABLE `reminder_month` (
	`user_id` text NOT NULL,
	`month` text NOT NULL,
	`sent` integer DEFAULT 0 NOT NULL,
	PRIMARY KEY(`user_id`, `month`)
);
