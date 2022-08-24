package database

import (
	"database/sql"

	"github.com/prestodb/presto-go-client/presto"
	_ "github.com/prestodb/presto-go-client/presto"
)

type Instance struct {
	db     *sql.DB
	config presto.Config
}

func New(config *presto.Config) (*Instance, error) {

	dsn, err := config.FormatDSN()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("presto", dsn)
	ii := &Instance{
		db:     db,
		config: *config,
	}
	return ii, err
}
