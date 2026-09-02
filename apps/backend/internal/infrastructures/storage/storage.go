// Package storage abstracts object storage backends. Business modules must
// depend on ObjectStorage instead of any concrete cloud SDK.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrObjectNotFound is returned by Head when the object does not exist.
var ErrObjectNotFound = errors.New("object not found")

type ObjectMetadata struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

type ObjectStorage interface {
	// PresignPut returns a time-limited URL that allows a PUT request for
	// exactly one object key with the given content type.
	PresignPut(
		ctx context.Context,
		key string,
		contentType string,
		expires time.Duration,
	) (string, error)

	Head(
		ctx context.Context,
		key string,
	) (*ObjectMetadata, error)

	Delete(
		ctx context.Context,
		key string,
	) error
}
