package sqlite_storage

import (
	"time"

	"github.com/humanbelnik/load-balancer/internal/ratelimiter/ratelimiter"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db *sqlx.DB
}

func New(path string) (*SQLiteStorage, error) {
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS clients (
		client_ip TEXT PRIMARY KEY,
		capacity INTEGER NOT NULL,
		refill_every INTEGER NOT NULL
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Add(clientID string, capacity int, refillEvery time.Duration) error {
	_, err := s.db.NamedExec(`
	INSERT INTO clients (client_ip, capacity, refill_every)
	VALUES (:client_ip, :capacity, :refill_every)
	ON CONFLICT(client_ip) DO UPDATE SET
		capacity = excluded.capacity,
		refill_every = excluded.refill_every;
	`, map[string]interface{}{
		"client_ip":    clientID,
		"capacity":     capacity,
		"refill_every": refillEvery.Nanoseconds(),
	})
	return err
}

func (s *SQLiteStorage) Delete(clientID string) error {
	_, err := s.db.Exec(`DELETE FROM clients WHERE client_ip = ?`, clientID)
	return err
}

func (s *SQLiteStorage) LoadAll() ([]ratelimiter.ClientConfig, error) {
	var clients []ratelimiter.ClientConfig
	err := s.db.Select(&clients, `SELECT client_ip, capacity, refill_every FROM clients`)
	for i := range clients {
		clients[i].RefillEvery = time.Duration(clients[i].RefillEvery)
	}
	return clients, err
}
