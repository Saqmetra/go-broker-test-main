CREATE TABLE IF NOT EXISTS trades_q (
                                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                                        account TEXT NOT NULL,
                                        symbol TEXT NOT NULL,
                                        volume REAL NOT NULL,
                                        open REAL NOT NULL,
                                        close REAL NOT NULL,
                                        side TEXT NOT NULL,
                                        processed BOOLEAN DEFAULT FALSE,
                                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS account_stats (
                                             account TEXT PRIMARY KEY,
                                             trades INTEGER DEFAULT 0,
                                             profit REAL DEFAULT 0.0
);