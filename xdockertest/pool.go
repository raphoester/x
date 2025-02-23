package xdockertest

import (
	"fmt"
	"time"

	"github.com/ory/dockertest"
)

func newPool() (*dockertest.Pool, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed creating dockertest pool: %w", err)
	}
	pool.MaxWait = 30 * time.Second
	return pool, nil
}
