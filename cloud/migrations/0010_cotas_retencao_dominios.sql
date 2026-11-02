CREATE TABLE `device_approval` (
	`code_hash` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`reservation_id` text NOT NULL,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `device_approval_expiry_idx` ON `device_approval` (`expires_at`);--> statement-breakpoint
CREATE TABLE `resource_quota` (
	`user_id` text PRIMARY KEY NOT NULL,
	`storage` integer DEFAULT 0 NOT NULL,
	`sites` integer DEFAULT 0 NOT NULL,
	`vaults` integer DEFAULT 0 NOT NULL,
	`ai` integer DEFAULT 0 NOT NULL,
	`devices` integer DEFAULT 0 NOT NULL,
	`deleted_at` integer,
	`purge_after` integer,
	`purge_started` integer,
	`grace_until` integer,
	`downgrade_notified` integer,
	`last_plan` text,
	`billing_cancel_pending` integer DEFAULT 0 NOT NULL
);
--> statement-breakpoint
CREATE INDEX `resource_purge_idx` ON `resource_quota` (`purge_after`);--> statement-breakpoint
CREATE INDEX `resource_grace_idx` ON `resource_quota` (`grace_until`);--> statement-breakpoint
CREATE INDEX `resource_billing_pending_idx` ON `resource_quota` (`billing_cancel_pending`);--> statement-breakpoint
CREATE INDEX `resource_notice_idx` ON `resource_quota` (`grace_until`) WHERE "resource_quota"."downgrade_notified" IS NULL AND "resource_quota"."deleted_at" IS NULL;--> statement-breakpoint
CREATE TABLE `resource_reservation` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`kind` text NOT NULL,
	`amount` integer NOT NULL,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `reservation_expiry_idx` ON `resource_reservation` (`expires_at`);--> statement-breakpoint
CREATE INDEX `reservation_user_idx` ON `resource_reservation` (`user_id`);--> statement-breakpoint
CREATE TABLE `retired_deploy` (
	`site_id` text NOT NULL,
	`deploy` text NOT NULL,
	`delete_after` integer NOT NULL,
	PRIMARY KEY(`site_id`, `deploy`)
);
--> statement-breakpoint
CREATE INDEX `retired_deploy_due_idx` ON `retired_deploy` (`delete_after`);--> statement-breakpoint
CREATE TABLE `site_domain` (
	`site_id` text PRIMARY KEY NOT NULL,
	`hostname` text NOT NULL,
	`proof` text NOT NULL,
	`verified_at` integer,
	`provider_id` text,
	`status` text DEFAULT 'pending_ownership' NOT NULL,
	`ssl` text DEFAULT 'pending' NOT NULL,
	`records` text DEFAULT '[]' NOT NULL,
	`checked_at` integer
);
--> statement-breakpoint
CREATE UNIQUE INDEX `site_domain_verified_unique` ON `site_domain` (`hostname`) WHERE "site_domain"."verified_at" IS NOT NULL;--> statement-breakpoint
CREATE TABLE `site_lease` (
	`site_id` text PRIMARY KEY NOT NULL,
	`token` text NOT NULL,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `site_lease_expiry_idx` ON `site_lease` (`expires_at`);--> statement-breakpoint
ALTER TABLE `session` ADD `studio_device` integer DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE `reminder` ADD `attempts` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `reminder` ADD `next_attempt_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `reminder` ADD `claim` text;--> statement-breakpoint
ALTER TABLE `reminder` ADD `claim_until` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `reminder_cleanup_idx` ON `reminder` (`send_at`);--> statement-breakpoint
ALTER TABLE `reminder_month` ADD `reserved` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `reminder_month_cleanup_idx` ON `reminder_month` (`month`);