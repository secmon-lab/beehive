package safe_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/safe"
)

type errCloser struct {
	closed bool
	err    error
}

func (e *errCloser) Close() error {
	e.closed = true
	return e.err
}

func TestClose_NilIsNoop(t *testing.T) {
	// Should not panic.
	safe.Close(context.Background(), nil)
}

func TestClose_CallsClose(t *testing.T) {
	c := &errCloser{}
	safe.Close(context.Background(), c)
	gt.True(t, c.closed)
}

func TestClose_LogsErrorButDoesNotPanic(t *testing.T) {
	c := &errCloser{err: errors.New("nope")}
	// We only assert that the call returns; the error is swallowed by
	// design (Close has no return value).
	safe.Close(context.Background(), c)
	gt.True(t, c.closed)
}

func TestCopy(t *testing.T) {
	src := strings.NewReader("hello")
	dst := &bytes.Buffer{}
	safe.Copy(context.Background(), dst, src)
	gt.Equal(t, dst.String(), "hello")
}
