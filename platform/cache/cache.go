// Package cache defines a portable key/value cache used by generated handlers
// and by the LLM gateway's response cache.
//
// Reference adapter (Redis) lives at cache/redis. The interface is small on
// purpose; richer features (versioning, secondary indexes, distributed locks)
// belong in dedicated subpackages, not here.
package cache

import (
	"context"
	"time"
)

// Cache is the abstract cache interface.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}
