package database

import (
	"database/sql"
	"fmt"
)

type Database interface {
	Connect(dsn string) (*sql.DB, error)
}

type OracleDatabase struct{}

func NewOracleDatabase() *OracleDatabase {
	return &OracleDatabase{}
}

func (o *OracleDatabase) Connect(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("db connection error: %w", err)
	}

	err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("db ping error: %w", err)
	}

	return conn, nil
}
