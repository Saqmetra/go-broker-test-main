package main

import (
	"database/sql"
	"flag"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Command line flags
	dbPath := flag.String("db", "data.db", "path to SQLite database")
	pollInterval := flag.Duration("poll", 100*time.Millisecond, "polling interval")
	flag.Parse()

	// Initialize database connection
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Worker started with polling interval: %v", *pollInterval)

	// Main worker loop
	for {
		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			time.Sleep(*pollInterval)
			continue
		}

		// Get unprocessed trade with FOR UPDATE lock
		var (
			id      int
			account string
			symbol  string
			volume  float64
			open    float64
			close   float64
			side    string
		)

		err = tx.QueryRow(`
			SELECT id, account, symbol, volume, open, close, side 
			FROM trades_q 
			WHERE processed = FALSE 
			ORDER BY created_at ASC 
			LIMIT 1 FOR UPDATE
		`).Scan(&id, &account, &symbol, &volume, &open, &close, &side)

		if err == sql.ErrNoRows {
			tx.Rollback()
			time.Sleep(*pollInterval)
			continue
		} else if err != nil {
			log.Printf("Failed to query trades: %v", err)
			tx.Rollback()
			time.Sleep(*pollInterval)
			continue
		}

		// Calculate profit
		lot := 100000.0
		profit := (close - open) * volume * lot
		if side == "sell" {
			profit = -profit
		}

		// Update account stats
		_, err = tx.Exec(`
			INSERT INTO account_stats (account, trades, profit) 
			VALUES (?, 1, ?)
			ON CONFLICT(account) DO UPDATE SET 
				trades = trades + 1,
				profit = profit + excluded.profit
		`, account, profit)

		if err != nil {
			log.Printf("Failed to update stats: %v", err)
			tx.Rollback()
			time.Sleep(*pollInterval)
			continue
		}

		// Mark trade as processed
		_, err = tx.Exec("UPDATE trades_q SET processed = TRUE WHERE id = ?", id)
		if err != nil {
			log.Printf("Failed to mark trade as processed: %v", err)
			tx.Rollback()
			time.Sleep(*pollInterval)
			continue
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			time.Sleep(*pollInterval)
			continue
		}

		log.Printf("Processed trade %d for account %s, profit: %.2f", id, account, profit)
		time.Sleep(*pollInterval)
	}
}
