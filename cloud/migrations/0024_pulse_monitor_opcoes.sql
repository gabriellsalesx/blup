ALTER TABLE `pulse_monitor` ADD `method` text DEFAULT 'GET' NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `expected_status` text DEFAULT '[]' NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `keyword` text;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `keyword_mode` text;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `timeout_seconds` integer DEFAULT 10 NOT NULL;--> statement-breakpoint
ALTER TABLE `pulse_monitor` ADD `headers` text;