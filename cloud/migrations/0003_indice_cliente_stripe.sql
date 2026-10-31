-- The webhook finds the account of a Stripe customer (billing_event). The
-- Stripe plugin adds the column but no index.
CREATE INDEX `user_stripe_customer_idx` ON `user` (`stripe_customer_id`);
