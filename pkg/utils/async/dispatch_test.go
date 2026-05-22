package async_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/async"
)

func TestDispatch_RunsFn(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	var ran bool
	async.Dispatch(context.Background(), func(ctx context.Context) error {
		defer wg.Done()
		ran = true
		return nil
	})
	wg.Wait()
	gt.True(t, ran)
}

func TestDispatch_RecoversPanic(t *testing.T) {
	done := make(chan struct{})
	async.Dispatch(context.Background(), func(ctx context.Context) error {
		defer close(done)
		panic("explode")
	})
	select {
	case <-done:
		// recovered without crashing the test process.
	case <-time.After(time.Second):
		t.Fatal("goroutine did not finish")
	}
}

func TestDispatch_HandlesError(t *testing.T) {
	done := make(chan struct{})
	async.Dispatch(context.Background(), func(ctx context.Context) error {
		defer close(done)
		return errors.New("expected")
	})
	select {
	case <-done:
		// errutil.Handle should have been invoked; we only assert the
		// goroutine returns normally.
	case <-time.After(time.Second):
		t.Fatal("goroutine did not finish")
	}
}
