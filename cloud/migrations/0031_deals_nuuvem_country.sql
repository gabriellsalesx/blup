ALTER TABLE `deals_post` ADD `price_shown` integer DEFAULT 1 NOT NULL;--> statement-breakpoint
-- CheapShark is no longer a source: its hunts go (with what they still had
-- queued) and so does its read; Nuuvem's sales take their place.
DELETE FROM `deals_post` WHERE `state`='queued' AND `hunt_id` IN (SELECT `id` FROM `deals_hunt` WHERE `kind`='games_deals');--> statement-breakpoint
DELETE FROM `deals_hunt` WHERE `kind`='games_deals';--> statement-breakpoint
DELETE FROM `deals_source` WHERE `id`='cheapshark';
