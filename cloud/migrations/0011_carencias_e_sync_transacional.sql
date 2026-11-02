CREATE TABLE `site_cache_purge` (
	`site_id` text PRIMARY KEY NOT NULL,
	`generation` text NOT NULL,
	`queued_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `site_cache_purge_queued_idx` ON `site_cache_purge` (`queued_at`);--> statement-breakpoint
CREATE TABLE `site_cache_purge_state` (
	`id` text PRIMARY KEY NOT NULL,
	`next_at` integer NOT NULL
);
--> statement-breakpoint
CREATE TABLE `sync_garbage` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`keys` text NOT NULL,
	`bytes` integer NOT NULL,
	`created_at` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `sync_garbage_user_idx` ON `sync_garbage` (`user_id`);--> statement-breakpoint
CREATE INDEX `sync_garbage_created_idx` ON `sync_garbage` (`created_at`);--> statement-breakpoint
CREATE TABLE `sync_transfer` (
	`id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`vault_id` text NOT NULL,
	`parent` integer NOT NULL,
	`key` text NOT NULL,
	`size` integer NOT NULL,
	`reservation_id` text NOT NULL,
	`reserved` integer NOT NULL,
	`lease` text NOT NULL,
	`expires_at` integer NOT NULL,
	`created_at` integer NOT NULL,
	`completed_version` integer
);
--> statement-breakpoint
CREATE INDEX `sync_transfer_expiry_idx` ON `sync_transfer` (`expires_at`);--> statement-breakpoint
CREATE INDEX `sync_transfer_user_idx` ON `sync_transfer` (`user_id`);--> statement-breakpoint
CREATE TABLE `sync_transfer_block` (
	`ticket_id` text NOT NULL,
	`block_id` text NOT NULL,
	`size` integer NOT NULL,
	`ready` integer DEFAULT 0 NOT NULL,
	`object_key` text,
	PRIMARY KEY(`ticket_id`, `block_id`)
);
--> statement-breakpoint
CREATE TABLE `vault_head_block` (
	`vault_id` text NOT NULL,
	`block_id` text NOT NULL,
	PRIMARY KEY(`vault_id`, `block_id`)
);
--> statement-breakpoint
CREATE INDEX `vault_head_block_id_idx` ON `vault_head_block` (`block_id`);--> statement-breakpoint
ALTER TABLE `resource_quota` ADD `storage_grace_limit` integer;--> statement-breakpoint
ALTER TABLE `resource_quota` ADD `site_cache_grace` integer;--> statement-breakpoint
ALTER TABLE `resource_quota` ADD `selected_site_id` text;--> statement-breakpoint
ALTER TABLE `resource_quota` ADD `domain_grace_until` integer;--> statement-breakpoint
ALTER TABLE `resource_quota` ADD `domain_downgrade_notified` integer;--> statement-breakpoint
CREATE INDEX `resource_site_cache_expiry_idx` ON `resource_quota` (`grace_until`) WHERE "resource_quota"."site_cache_grace" IS NOT "resource_quota"."grace_until" AND "resource_quota"."deleted_at" IS NULL;--> statement-breakpoint
CREATE INDEX `resource_domain_notice_idx` ON `resource_quota` (`domain_grace_until`) WHERE "resource_quota"."domain_downgrade_notified" IS NULL AND "resource_quota"."deleted_at" IS NULL;--> statement-breakpoint
ALTER TABLE `site_domain` ADD `disable_at` integer;--> statement-breakpoint
ALTER TABLE `site_domain` ADD `disabled_at` integer;--> statement-breakpoint
CREATE INDEX `site_domain_disable_idx` ON `site_domain` (`disable_at`) WHERE "site_domain"."status" <> 'disabled' OR "site_domain"."provider_id" IS NOT NULL OR "site_domain"."verified_at" IS NOT NULL;--> statement-breakpoint
ALTER TABLE `sync_block` ADD `object_key` text;--> statement-breakpoint
ALTER TABLE `vault` ADD `head_refs_ready` integer DEFAULT false NOT NULL;