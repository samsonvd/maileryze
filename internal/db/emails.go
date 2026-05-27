package db

import (
	"database/sql"
	"fmt"
	"time"
)

type EmailRecord struct {
	ID                 int
	Subject            string
	ReceivedAt         time.Time
	ProviderIdentifier string
	UnsubscribeURL     string
	UnsubscribeEmail   string
}

// GetEmailsBySender returns all emails from a sender for an alias,
// ordered by received date descending. Used by the detail view.
func GetEmailsBySender(db *sql.DB, alias, senderAddress string) ([]EmailRecord, error) {
	rows, err := db.Query(`
		SELECT id, subject, received_at, provider_identifier,
		       COALESCE(unsubscribe_url, ''), COALESCE(unsubscribe_email, '')
		FROM emails
		WHERE alias = ? AND sender_address = ?
		ORDER BY received_at DESC`,
		alias, senderAddress)
	if err != nil {
		return nil, fmt.Errorf("querying emails by sender: %w", err)
	}
	defer rows.Close()

	var records []EmailRecord
	for rows.Next() {
		var r EmailRecord
		var receivedAt string
		if err := rows.Scan(&r.ID, &r.Subject, &receivedAt, &r.ProviderIdentifier, &r.UnsubscribeURL, &r.UnsubscribeEmail); err != nil {
			return nil, err
		}
		r.ReceivedAt, _ = parseTime(receivedAt)
		records = append(records, r)
	}
	return records, nil
}
