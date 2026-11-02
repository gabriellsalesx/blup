ALTER TABLE `pulse_incident` ADD `expires_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `pulse_incident_expiry_idx` ON `pulse_incident` (`expires_at`);--> statement-breakpoint
ALTER TABLE `pulse_incident_update` ADD `expires_at` integer DEFAULT 0 NOT NULL;--> statement-breakpoint
CREATE INDEX `pulse_incident_update_expiry_idx` ON `pulse_incident_update` (`expires_at`);