package database

import (
	"fmt"
	"log"

	dbutils "rpc_watcher/rpcwatcher/dbutils"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	cnsmodels "github.com/emerishq/demeris-backend-models/cns"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type Instance struct {
	d          *dbutils.Instance
	connString string
}

func New(connString string) (*Instance, error) {
	i, err := dbutils.New(connString)
	if err != nil {
		return nil, err
	}

	ii := &Instance{
		d:          i,
		connString: connString,
	}

	return ii, nil
}

func (i *Instance) UpdateDenoms(chain cnsmodels.Chain) error {
	//spew.Dump(chain)
	n, err := i.d.DB.PrepareNamed(`UPDATE cns.chains 
	SET denoms=:denoms 
	WHERE chain_name=:chain_name;`)
	if err != nil {
		return err
	}

	defer func() {
		err := n.Close()
		if err != nil {
			panic(err)
		}
	}()

	res, err := n.Exec(chain)

	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()

	if rows == 0 {
		return fmt.Errorf("database update statement had no effect")
	}

	return nil
}

func (i *Instance) Chain(chain string) (cnsmodels.Chain, error) {
	var c cnsmodels.Chain

	err := i.d.DB.Get(&c, fmt.Sprintf("SELECT * FROM cns.chains WHERE chain_name='%s' limit 1;", chain))

	return c, err
}

func (i *Instance) Chains() ([]cnsmodels.Chain, error) {
	var c []cnsmodels.Chain
	//spew.Dump(c)
	return c, i.d.Exec("SELECT * FROM cns.chains", nil, &c)
}

func (i *Instance) GetCounterParty(chain, srcChannel string) ([]cnsmodels.ChannelQuery, error) {
	var c []cnsmodels.ChannelQuery

	q, err := i.d.DB.PrepareNamed("select chain_name, json_data.* from cns.chains, jsonb_each_text(primary_channel) as json_data where chain_name=:chain_name and value=:channel limit 1;")
	if err != nil {
		return []cnsmodels.ChannelQuery{}, err
	}

	defer func() {
		err := q.Close()
		if err != nil {
			panic(err)
		}
	}()

	if err := q.Select(&c, map[string]interface{}{
		"chain_name": chain,
		"channel":    srcChannel,
	}); err != nil {
		return []cnsmodels.ChannelQuery{}, err
	}

	if len(c) == 0 {
		return nil, fmt.Errorf("no counterparty found for chain %s on channel %s", chain, srcChannel)
	}

	return c, nil
}

func SetupTestDB(migrations []string) (testserver.TestServer, *Instance) {
	// start new cockroachDB test server
	ts, err := testserver.NewTestServer()
	checkNoError(err)

	err = ts.WaitForInit()
	checkNoError(err)

	// create new instance of db
	i, err := New(ts.PGURL().String())
	checkNoError(err)

	// create and insert data into db
	err = dbutils.RunMigrations(ts.PGURL().String(), migrations)
	checkNoError(err)

	return ts, i
}

func checkNoError(err error) {
	if err != nil {
		log.Fatalf("got error: %s", err)
	}
}
