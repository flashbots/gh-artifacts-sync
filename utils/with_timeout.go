package utils

import (
	"context"
	"time"
)

func WithTimeout[T any](
	parent context.Context,
	timeout time.Duration,
	do func(ctx context.Context) (T, error),
) (T, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	return do(ctx)
}
