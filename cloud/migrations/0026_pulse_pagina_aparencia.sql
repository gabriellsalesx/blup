ALTER TABLE `pulse_page` ADD `show_title` integer DEFAULT 1 NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_page` ADD `nav_links` text DEFAULT '[]' NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_page` ADD `logo_light` text;--> statement-breakpoint
ALTER TABLE `pulse_page` ADD `logo_dark` text;