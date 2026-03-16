CREATE TABLE IF NOT EXISTS "VpnUser" (
    username TEXT PRIMARY KEY,
    telegramId BIGINT NOT NULL,
    accountDetailsLink TEXT NOT NULL,
    "isUsedTestPeriod" BOOLEAN NOT NULL DEFAULT false
);
