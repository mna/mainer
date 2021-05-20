package mainer

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestCurrentStdio(t *testing.T) {
	c := qt.New(t)

	cwd, err := os.Getwd()
	c.Assert(err, qt.IsNil)
	c.Assert(cwd, qt.Equals, CurrentStdio().Cwd)
}

func TestCancelOnSignal(t *testing.T) {
	c := qt.New(t)

	ctx := context.Background()
	ctx = CancelOnSignal(ctx, syscall.SIGUSR1)

	select {
	case <-ctx.Done():
		c.Fatal("context should block")
	default:
	}

	proc, err := os.FindProcess(os.Getpid())
	c.Assert(err, qt.IsNil)
	err = proc.Signal(syscall.SIGUSR1)
	c.Assert(err, qt.IsNil)

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		c.Fatal("context should be done")
	}
}

func TestCancelOnSignal_NoSignal(t *testing.T) {
	c := qt.New(t)

	ctx := context.Background()
	ctx2 := CancelOnSignal(ctx)
	c.Assert(ctx, qt.Equals, ctx2)
}
