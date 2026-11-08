-- O canal de uma sessão da Ponte (WhatsApp), achado pelo aviso dela.
-- Os índices de user e subscription (0001 e 0003) entraram agora no snapshot
-- por worker/db/auth-indexes.ts; já existem no banco, então não vão aqui.
CREATE INDEX `deals_channel_session_idx` ON `deals_channel` (`session_id`) WHERE "deals_channel"."session_id" IS NOT NULL;
