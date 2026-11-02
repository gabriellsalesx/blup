CREATE TABLE `billing_checkout` (
	`user_id` text PRIMARY KEY NOT NULL,
	`id` text NOT NULL,
	`plan` text NOT NULL,
	`annual` integer NOT NULL,
	`params` text NOT NULL,
	`session_id` text,
	`url` text,
	`expires_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `billing_checkout_expiry_idx` ON `billing_checkout` (`expires_at`);--> statement-breakpoint
CREATE TABLE `hostname_claim` (
	`hostname` text PRIMARY KEY NOT NULL,
	`kind` text NOT NULL,
	`resource_id` text NOT NULL
);
--> statement-breakpoint
CREATE TABLE `pulse_billing_ticket` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`subscription_id` text NOT NULL,
	`target_plan` text NOT NULL,
	`annual` integer NOT NULL,
	`proration_date` integer NOT NULL,
	`fingerprint` text NOT NULL,
	`quote` text NOT NULL,
	`target_price` text,
	`expires_at` integer NOT NULL,
	`state` text DEFAULT 'preview' NOT NULL,
	`result` text,
	`schedule_id` text
);
--> statement-breakpoint
CREATE INDEX `pulse_billing_ticket_owner_idx` ON `pulse_billing_ticket` (`user_id`);--> statement-breakpoint
CREATE INDEX `pulse_billing_ticket_expiry_idx` ON `pulse_billing_ticket` (`expires_at`);--> statement-breakpoint
CREATE TABLE `pulse_selection` (
	`user_id` text PRIMARY KEY NOT NULL,
	`target_plan` text NOT NULL,
	`effective_at` integer NOT NULL,
	`keep` text NOT NULL,
	`applied_at` integer
);
--> statement-breakpoint
CREATE INDEX `pulse_selection_due_idx` ON `pulse_selection` (`applied_at`,`effective_at`);--> statement-breakpoint
ALTER TABLE `pulse_account` ADD `paid_until` integer;--> statement-breakpoint
ALTER TABLE `pulse_account` ADD `resource_plan` text;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `plan_disabled` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `retire_at` integer;--> statement-breakpoint
CREATE INDEX `pulse_monitor_retire_idx` ON `pulse_monitor` (`retire_at`);--> statement-breakpoint
ALTER TABLE `pulse_page` ADD `plan_disabled` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_page` ADD `retire_at` integer;--> statement-breakpoint
CREATE INDEX `pulse_page_retire_idx` ON `pulse_page` (`retire_at`);--> statement-breakpoint
ALTER TABLE `pulse_project` ADD `plan_disabled` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_project` ADD `retire_at` integer;--> statement-breakpoint
CREATE INDEX `pulse_project_retire_idx` ON `pulse_project` (`retire_at`);