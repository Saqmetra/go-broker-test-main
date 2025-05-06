package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

type Trade struct {
	Account string  `json:"account"`
	Symbol  string  `json:"symbol"`
	Volume  float64 `json:"volume"`
	Open    float64 `json:"open"`
	Close   float64 `json:"close"`
	Side    string  `json:"side"`
}

type Stats struct {
	Account string  `json:"account"`
	Trades  int     `json:"trades"`
	Profit  float64 `json:"profit"`
}

func main() {
	// Command line flags
	dbPath := flag.String("db", "data.db", "path to SQLite database")
	listenAddr := flag.String("listen", "8080", "HTTP server listen address")
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

	// Initialize HTTP server
	mux := http.NewServeMux()

	// POST /trades endpoint
	mux.HandleFunc("POST /trades", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var trade Trade
		if err := json.NewDecoder(r.Body).Decode(&trade); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if trade.Account == "" {
			http.Error(w, "Account must not be empty", http.StatusBadRequest)
			return
		}

		symbolRegex := regexp.MustCompile(`^[A-Z]{6}$`)
		if !symbolRegex.MatchString(trade.Symbol) {
			http.Error(w, "Symbol must be 6 uppercase letters", http.StatusBadRequest)
			return
		}

		if trade.Volume <= 0 || trade.Open <= 0 || trade.Close <= 0 {
			http.Error(w, "Volume, open and close must be positive", http.StatusBadRequest)
			return
		}

		if trade.Side != "buy" && trade.Side != "sell" {
			http.Error(w, "Side must be either 'buy' or 'sell'", http.StatusBadRequest)
			return
		}
		
		_, err := db.Exec(
			"INSERT INTO trades_q (account, symbol, volume, open, close, side) VALUES (?, ?, ?, ?, ?, ?)",
			trade.Account, trade.Symbol, trade.Volume, trade.Open, trade.Close, trade.Side,
		)

		if err != nil {
			http.Error(w, "Failed to enqueue trade", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// GET /stats/{acc} endpoint
	mux.HandleFunc("GET /stats/{acc}", func(w http.ResponseWriter, r *http.Request) {

		account := r.PathValue("acc")

		if account == "" {
			http.Error(w, "Account must not be empty", http.StatusBadRequest)
			return
		}

		var stats Stats

		err := db.QueryRow(
			"SELECT account, trades, profit FROM account_stats WHERE account = ?",
			account,
		).Scan(&stats.Account, &stats.Trades, &stats.Profit)

		if err == sql.ErrNoRows {
			stats = Stats{Account: account, Trades: 0, Profit: 0.0}
		} else if err != nil {
			http.Error(w, "Failed to get stats", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// GET /healthz endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {

		if err := db.Ping(); err != nil {
			http.Error(w, "Database not available", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Start server
	serverAddr := fmt.Sprintf(":%s", *listenAddr)
	log.Printf("Starting server on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
