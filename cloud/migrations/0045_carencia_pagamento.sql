-- Carência de 14 dias do pagamento falho: quando a assinatura ficou past_due.
CREATE TABLE `subscription_grace` (
	`subscription_id` text PRIMARY KEY NOT NULL,
	`user_id` text NOT NULL,
	`past_due_since` integer NOT NULL
);
--> statement-breakpoint
CREATE INDEX `subscription_grace_since_idx` ON `subscription_grace` (`past_due_since`);--> statement-breakpoint
CREATE INDEX `subscription_grace_user_idx` ON `subscription_grace` (`user_id`);--> statement-breakpoint
-- As que já estão past_due contam a carência a partir de agora.
INSERT INTO `subscription_grace` (`subscription_id`,`user_id`,`past_due_since`) SELECT `id`,`reference_id`,CAST(strftime('%s','now') AS INTEGER)*1000 FROM `subscription` WHERE `status`='past_due';
