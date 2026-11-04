CREATE TABLE `deals_account` (
	`user_id` text PRIMARY KEY NOT NULL,
	`last_plan` text NOT NULL,
	`read_only_until` integer,
	`purge_started` integer
);
--> statement-breakpoint
CREATE INDEX `deals_account_read_only_idx` ON `deals_account` (`read_only_until`);