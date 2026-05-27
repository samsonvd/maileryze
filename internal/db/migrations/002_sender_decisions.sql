-- +goose Up
CREATE TABLE sender_decisions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    alias           TEXT NOT NULL,
    sender_address  TEXT NOT NULL,
    decision        TEXT NOT NULL CHECK(decision IN ('keep', 'deleted', 'unsubscribed', 'unsubscribed_deleted', 'skipped')),
    emails_affected INTEGER NOT NULL DEFAULT 0,
    decided_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(alias, sender_address)
);

-- +goose Down
DROP TABLE sender_decisions;
