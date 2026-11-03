-- Every alert destination belongs to a project: the ones no project sends
-- to (replaced or taken off before this rule) leave, with their queue and
-- cooldowns.
DELETE FROM `pulse_outbox` WHERE `channel_id` NOT IN (SELECT `channel_id` FROM `pulse_project_channel`);--> statement-breakpoint
DELETE FROM `pulse_cooldown` WHERE substr(`key`,1,instr(`key`,':')-1) IN (SELECT `id` FROM `pulse_channel` WHERE `id` NOT IN (SELECT `channel_id` FROM `pulse_project_channel`));--> statement-breakpoint
DELETE FROM `pulse_channel` WHERE `id` NOT IN (SELECT `channel_id` FROM `pulse_project_channel`);
