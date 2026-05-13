-- +goose Up
-- SQLite doesn't support ALTER TABLE ADD CONSTRAINT, so recreate the table
-- to add alias and update the unique constraint to include it.
CREATE TABLE emails_new (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    alias               TEXT NOT NULL DEFAULT '',
    subject             TEXT NOT NULL,
    sender_name         TEXT NOT NULL,
    sender_address      TEXT NOT NULL,
    unsubscribe_email   TEXT,
    unsubscribe_url     TEXT,
    provider            TEXT NOT NULL,
    provider_identifier TEXT NOT NULL,
    received_at         DATETIME NOT NULL,
    fetched_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (alias, provider, provider_identifier)
);

INSERT INTO emails_new
    SELECT id, '', subject, sender_name, sender_address, unsubscribe_email, unsubscribe_url, provider, provider_identifier, received_at, fetched_at
    FROM emails;

DROP TABLE emails;
ALTER TABLE emails_new RENAME TO emails;

-- +goose Down
CREATE TABLE emails_old (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    subject             TEXT NOT NULL,
    sender_name         TEXT NOT NULL,
    sender_address      TEXT NOT NULL,
    unsubscribe_email   TEXT,
    unsubscribe_url     TEXT,
    provider            TEXT NOT NULL,
    provider_identifier TEXT NOT NULL,
    received_at         DATETIME NOT NULL,
    fetched_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (provider, provider_identifier)
);

INSERT OR IGNORE INTO emails_old
    SELECT id, subject, sender_name, sender_address, unsubscribe_email, unsubscribe_url, provider, provider_identifier, received_at, fetched_at
    FROM emails;

DROP TABLE emails;
ALTER TABLE emails_old RENAME TO emails;
