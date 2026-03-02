CREATE TABLE IF NOT EXISTS "invoice_status" (
    id SERIAL PRIMARY KEY,
    invoice_id BIGINT NOT NULL,
    "telegramId" BIGINT NOT NULL,
    status TEXT NOT NULL,
    payload TEXT
);
