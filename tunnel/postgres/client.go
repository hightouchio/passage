package postgres

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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
