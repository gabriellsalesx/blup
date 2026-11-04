CREATE TABLE `ai_model_setting` (
	`feature` text PRIMARY KEY NOT NULL,
	`models` text NOT NULL,
	`updated_by` text NOT NULL,
	`updated_at` integer NOT NULL
);
--> statement-breakpoint
ALTER TABLE `deals_post` ADD `description` text;--> statement-breakpoint
ALTER TABLE `deals_post` ADD `coupon` text;--> statement-breakpoint
ALTER TABLE `deals_setting` ADD `style` text DEFAULT 'classic' NOT NULL;