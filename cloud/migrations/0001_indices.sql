-- Indexes the queries need (checked with EXPLAIN QUERY PLAN). D1 bills
-- every row a query reads; without these, each one scans its whole table.

-- The account's subscription (/api/me on every page, the staff, the
-- overview): by account and status. The Stripe plugin doesn't create it.
CREATE INDEX `subscription_reference_status_idx` ON `subscription` (`reference_id`, `status`);
--> statement-breakpoint
-- The staff overview: accounts created in the last 7 days.
CREATE INDEX `user_created_at_idx` ON `user` (`created_at`);
--> statement-breakpoint
-- The staff overview: blocked accounts (few, so a partial index).
CREATE INDEX `user_banned_idx` ON `user` (`banned`) WHERE `banned` = 1;
