package db

import (
	"database/sql"
	"fmt"
	"time"
)

type SenderDecision struct {
	Alias          string
	SenderAddress  string
	Decision       string
	EmailsAffected int
	DecidedAt      time.Time
}

// SaveDecision upserts a triage decision for a sender.
// decision must be one of: keep, deleted, unsubscribed, unsubscribed_deleted, skipped.
func SaveDecision(db *sql.DB, alias, senderAddress, decision string, emailsAffected int) error {
	_, err := db.Exec(`
		INSERT INTO sender_decisions (alias, sender_address, decision, emails_affected)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(alias, sender_address) DO UPDATE SET
			decision        = excluded.decision,
			emails_affected = excluded.emails_affected,
			decided_at      = CURRENT_TIMESTAMP`,
		alias, senderAddress, decision, emailsAffected,
	)
	return err
}

// GetDecisions returns all decisions for an alias, keyed by sender_address.
func GetDecisions(db *sql.DB, alias string) (map[string]SenderDecision, error) {
	rows, err := db.Query(`
		SELECT alias, sender_address, decision, emails_affected, decided_at
		FROM sender_decisions
		WHERE alias = ?`, alias)
	if err != nil {
		return nil, fmt.Errorf("querying decisions: %w", err)
	}
	defer rows.Close()

	decisions := make(map[string]SenderDecision)
	for rows.Next() {
		var d SenderDecision
		var decidedAt string
		if err := rows.Scan(&d.Alias, &d.SenderAddress, &d.Decision, &d.EmailsAffected, &decidedAt); err != nil {
			return nil, err
		}
		d.DecidedAt, _ = parseTime(decidedAt)
		decisions[d.SenderAddress] = d
	}
	return decisions, nil
}

// GetDecisionCounts returns the count of decisions per type for an alias.
func GetDecisionCounts(db *sql.DB, alias string) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT decision, COUNT(*)
		FROM sender_decisions
		WHERE alias = ?
		GROUP BY decision`, alias)
	if err != nil {
		return nil, fmt.Errorf("querying decision counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, err
		}
		counts[decision] = count
	}
	return counts, nil
}
