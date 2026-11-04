ALTER TABLE `deals_channel` ADD `language` text;--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `short_title` text;--> statement-breakpoint
ALTER TABLE `deals_offer` ADD `category` text;--> statement-breakpoint
ALTER TABLE `deals_setting` ADD `ai_titles` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
ALTER TABLE `deals_setting` ADD `ai_describe` integer DEFAULT 0 NOT NULL;