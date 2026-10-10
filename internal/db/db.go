package db

import (
	"database/sql"

	_ "github.com/lib/pq"
	"novel-be/config"
)

func Open(cfg config.Config) (*sql.DB, error) {
	return sql.Open("postgres", cfg.GetConnectionString())
}
