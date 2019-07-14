CREATE TABLE IF NOT EXISTS zerochain.txn_summary (
hash text PRIMARY KEY,
round bigint
);

CREATE INDEX IF NOT EXISTS ON zerochain.txn_summary(round);
