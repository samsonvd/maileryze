-- +goose Up
CREATE TABLE emails (
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

-- +goose Down
DROP TABLE emails;
