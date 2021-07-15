package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Client struct {
	db *sqlx.DB
}

func NewClient(db *sqlx.DB) Client {
	return Client{db}
}

type scanner interface {
	Scan(...interface{}) error
}

var ErrTunnelNotFound = errors.New("tunnel not found")
