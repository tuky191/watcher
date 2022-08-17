package store

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrBlockNotFound = fmt.Errorf("block not found")

const (
	blockFmt       = "block/%d"
	blockTimeFmt   = "blockTime/%d"
	defaultTimeout = 100 * 10 * time.Second // we keep the last 100 blocks, assuming block time of 10 seconds
)

type Blocks struct {
	storeInstance *Store
}

func NewBlocks(s *Store) *Blocks {
	return &Blocks{storeInstance: s}
}

func blockKey(height int64) string {
	return fmt.Sprintf(blockFmt, height)
}

func blockTimeKey(height int64) string {
	return fmt.Sprintf(blockTimeFmt, height)
}

func (b *Blocks) queryRedis(key string) ([]byte, error) {
	res, err := b.storeInstance.Client.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrBlockNotFound
		}

		return nil, fmt.Errorf("redis error, %w", err)
	}

	return []byte(res), nil
}

func (b *Blocks) Block(height int64) ([]byte, error) {
	return b.queryRedis(blockKey(height))
}

func (b *Blocks) SetLastBlockTime(t time.Time, height int64) error {
	// TODO: figure out how to get block time from block
	mt, err := t.MarshalText()
	if err != nil {
		return err
	}
	return b.storeInstance.Client.Set(context.Background(), blockTimeKey(height), mt, defaultTimeout).Err()
}

func (b *Blocks) LastBlockTime(height int64) (time.Time, error) {
	res, err := b.queryRedis(blockTimeKey(height))
	if err != nil {
		return time.Time{}, err
	}

	tt := time.Time{}
	return tt, tt.UnmarshalText(res)
}

func (b *Blocks) Add(data []byte, height int64) error {
	//spew.Dump(string(data))
	return b.storeInstance.Client.Set(context.Background(), blockKey(height), string(data), defaultTimeout).Err()
}
