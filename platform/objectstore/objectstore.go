// Package objectstore is a portable blob storage interface.
//
// Reference adapter (S3/Minio) lives at objectstore/s3. SignedURL is the
// boundary that lets generated handlers offload uploads/downloads to clients
// without proxying bytes through the application.
package objectstore

import (
	"context"
	"io"
	"time"
)

// Op is a signed-URL operation.
type Op int

const (
	OpGet Op = iota
	OpPut
)

// Meta carries the small set of object attributes vibeguard plumbs through.
type Meta struct {
	ContentType string
	Size        int64
	ETag        string
}

// Bucket is a single named blob namespace.
type Bucket interface {
	Put(ctx context.Context, key string, r io.Reader, meta Meta) error
	Get(ctx context.Context, key string) (io.ReadCloser, Meta, error)
	SignedURL(ctx context.Context, key string, op Op, ttl time.Duration) (string, error)
}
