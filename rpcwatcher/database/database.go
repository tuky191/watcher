package database

import (
	"database/sql"

	"github.com/prestodb/presto-go-client/presto"
	_ "github.com/prestodb/presto-go-client/presto"
)

type Instance struct {
	Handle *sql.DB
	config presto.Config
}

func New(config *presto.Config) (*Instance, error) {

	dsn, err := config.FormatDSN()
	if err != nil {
		return nil, err
	}
	//spew.Dump(dsn)
	db, err := sql.Open("presto", dsn)
	ii := &Instance{
		Handle: db,
		config: *config,
	}
	return ii, err
}
