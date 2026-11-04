CREATE TABLE `deals_setting` (
	`user_id` text PRIMARY KEY NOT NULL,
	`pace` integer DEFAULT 0 NOT NULL,
	`pace_last` integer DEFAULT 0 NOT NULL,
	`dropped_at` integer,
	`headlines` text DEFAULT '{}' NOT NULL,
	`extra` text,
	`extra_on` integer DEFAULT 0 NOT NULL,
	`socials` text DEFAULT '[]' NOT NULL,
	`socials_on` integer DEFAULT 0 NOT NULL,
	`updated_at` integer DEFAULT 0 NOT NULL
);
--> statement-breakpoint
ALTER TABLE `deals_post` ADD `paced` integer DEFAULT 0 NOT NULL;