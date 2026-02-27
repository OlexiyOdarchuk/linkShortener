package database

import (
	"database/sql"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type Analytics struct {
	db *sql.DB
}

func ConnectClickHouse(addr, user, pass, dbName string) (*Analytics, error) {
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: user,
			Password: pass,
		},
		DialTimeout: time.Second * 30,
	})
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &Analytics{db: conn}, nil
}
