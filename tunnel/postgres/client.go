package postgres

import "database/sql"

type Client struct {
	db *sql.DB
}

func NewClient(db *sql.DB) Client {
	return Client{db}
}

type scanner interface {
	Scan(...interface{}) error
}
