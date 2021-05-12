package postgres

import (
	"database/sql"
	"github.com/pkg/errors"
)

type Client struct {
	db *sql.DB
}

func NewClient(db *sql.DB) Client {
	return Client{db}
}

type scanner interface {
	Scan(...interface{}) error
}

var ErrTunnelNotFound = errors.New("tunnel not found")
