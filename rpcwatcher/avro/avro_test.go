package avro

import (
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/linkedin/goavro/v2"
	"github.com/stretchr/testify/require"
	"github.com/terra-money/mantlemint/block_feed"
)

type SliceStruct struct {
	ID   int
	Name string
}

type SimpleRecord struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Inline struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}
type SimpleRecordInline struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Inline Inline `json:"inline"`
}

type SimpleRecordSlice struct {
	ID            int            `json:"id"`
	SimpleRecords []SimpleRecord `json:"simple_records"`
}

const (
	BlockfeedBlockResult     = `{"type":"record","name":"BlockResult","fields":[{"name":"block_id","type":{"type":"record","name":"block_id","namespace":"BlockResult","fields":[{"name":"hash","type":"string"},{"name":"parts","type":{"type":"record","name":"parts","namespace":"BlockResult.block_id","fields":[{"name":"total","type":"int"},{"name":"hash","type":"string"}]}}]}},{"name":"block","type":{"type":"record","name":"block","namespace":"BlockResult","fields":[{"name":"mtx","type":{"type":"record","name":"mtx","namespace":"BlockResult.block","fields":[{"name":"Mutex","type":{"type":"record","name":"Mutex","namespace":"BlockResult.block.mtx","fields":[{"name":"state","type":"int"},{"name":"sema","type":"int"}]}}]}},{"name":"header","type":{"type":"record","name":"header","namespace":"BlockResult.block","fields":[{"name":"version","type":{"type":"record","name":"version","namespace":"BlockResult.block.header","fields":[{"name":"block","type":"long"},{"name":"app","type":"long"}]}},{"name":"chain_id","type":"string"},{"name":"height","type":"long"},{"name":"time","type":"string"},{"name":"last_block_id","type":{"type":"record","name":"last_block_id","namespace":"BlockResult.block.header","fields":[{"name":"hash","type":"string"},{"name":"parts","type":{"type":"record","name":"parts","namespace":"BlockResult.block.header.last_block_id","fields":[{"name":"total","type":"int"},{"name":"hash","type":"string"}]}}]}},{"name":"last_commit_hash","type":"string"},{"name":"data_hash","type":"string"},{"name":"validators_hash","type":"string"},{"name":"next_validators_hash","type":"string"},{"name":"consensus_hash","type":"string"},{"name":"app_hash","type":"string"},{"name":"last_results_hash","type":"string"},{"name":"evidence_hash","type":"string"},{"name":"proposer_address","type":"string"}]}},{"name":"data","type":{"type":"record","name":"data","namespace":"BlockResult.block","fields":[{"name":"txs","type":["null",{"type":"array","name":"Tx","namespace":"BlockResult.block.data","items":"string"}]},{"name":"hash","type":"string"}]}},{"name":"evidence","type":{"type":"record","name":"evidence","namespace":"BlockResult.block","fields":[{"name":"evidence","type":"string"},{"name":"hash","type":"string"},{"name":"byteSize","type":"long"}]}},{"name":"last_commit","type":{"type":"record","name":"last_commit","namespace":"BlockResult.block","fields":[{"name":"height","type":"long"},{"name":"round","type":"int"},{"name":"block_id","type":{"type":"record","name":"block_id","namespace":"BlockResult.block.last_commit","fields":[{"name":"hash","type":"string"},{"name":"parts","type":{"type":"record","name":"parts","namespace":"BlockResult.block.last_commit.block_id","fields":[{"name":"total","type":"int"},{"name":"hash","type":"string"}]}}]}},{"name":"signatures","type":["null",{"type":"array","name":"CommitSig","namespace":"BlockResult.block.last_commit","items":{"type":"record","name":"CommitSig","namespace":"BlockResult.block.last_commit","fields":[{"name":"block_id_flag","type":"int"},{"name":"validator_address","type":"string"},{"name":"timestamp","type":"string"},{"name":"signature","type":"string"}]}}]},{"name":"hash","type":"string"},{"name":"bitArray","type":{"type":"record","name":"bitArray","namespace":"BlockResult.block.last_commit","fields":[{"name":"mtx","type":{"type":"record","name":"mtx","namespace":"BlockResult.block.last_commit.bitArray","fields":[{"name":"state","type":"int"},{"name":"sema","type":"int"}]}},{"name":"bits","type":"int"},{"name":"elems","type":"string"}]}}]}}]}}]}`
	TxResult                 = `{"type":"record","name":"TxResult","fields":[{"name":"height","type":"long"},{"name":"index","type":"int"},{"name":"tx","type":"string"},{"name":"result","type":{"type":"record","name":"result","namespace":"TxResult","fields":[{"name":"code","type":"int"},{"name":"data","type":"string"},{"name":"log","type":"string"},{"name":"info","type":"string"},{"name":"gas_wanted","type":"long"},{"name":"gas_used","type":"long"},{"name":"events","type":{"type":"array","name":"Event","namespace":"TxResult.result","items":{"type":"record","name":"Event","namespace":"TxResult.result","fields":[{"name":"type","type":"string"},{"name":"attributes","type":{"type":"array","name":"EventAttribute","namespace":"TxResult.result.Event","items":{"type":"record","name":"EventAttribute","namespace":"TxResult.result.Event","fields":[{"name":"key","type":"string"},{"name":"value","type":"string"},{"name":"index","type":"boolean"}]}}}]}}},{"name":"codespace","type":"string"}]}}]}`
	SimpleRecordSchema       = `{"type":"record","name":"SimpleRecord","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`
	SimpleRecordInlineSchema = `{"type":"record","name":"SimpleRecordInline","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"inline","type":{"type":"record","name":"inline","namespace":"SimpleRecordInline","fields":[{"name":"id","type":"int"},{"name":"message","type":"string"}]}}]}`
	SimpleRecordSliceSchema  = `{"type":"record","name":"SimpleRecordSlice","fields":[{"name":"id","type":"int"},{"name":"simple_records","type":["null",{"type":"array","name":"SimpleRecord","namespace":"SimpleRecordSlice","items":{"type":"record","name":"SimpleRecord","namespace":"SimpleRecordSlice","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}}]}]}`
)

var (
	ComplexStructsMap = map[string]interface{}{
		BlockfeedBlockResult: &block_feed.BlockResult{},
		//	TxResult:             abci.TxResult{},
	}
	SimpleStructsMap = map[string]interface{}{
		SimpleRecordSliceSchema:  SimpleRecordSlice{},
		SimpleRecordSchema:       SimpleRecord{},
		SimpleRecordInlineSchema: SimpleRecordInline{},
	}
)

func TestSimpleStructsSchemaGeneration(t *testing.T) {
	for correct_schema, tested_struct := range SimpleStructsMap {
		schema, err := GenerateAvroSchema(tested_struct)
		require.NoError(t, err)
		//validate schema syntax
		_, err = goavro.NewCodec(schema)
		require.NoError(t, err)
		assert.Equal(t, schema, correct_schema)
	}

}
func TestComplexStructsSchemaGeneration(t *testing.T) {
	for correct_schema, tested_struct := range ComplexStructsMap {
		schema, err := GenerateAvroSchema(tested_struct)
		require.NoError(t, err)
		//validate schema syntax
		_, err = goavro.NewCodec(schema)
		require.NoError(t, err)
		assert.Equal(t, schema, correct_schema)
	}

}
