package plugin

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ExitWhenOrphaned(t *testing.T) {
	var ppid atomic.Int64
	ppid.Store(100)

	exited := make(chan struct{})

	go exitWhenOrphaned(100, func() int { return int(ppid.Load()) }, time.Millisecond, func() { close(exited) })

	select {
	case <-exited:
		t.Fatal("exited while parent was still alive")
	case <-time.After(20 * time.Millisecond):
	}

	// Parent died, so we've been re-parented
	ppid.Store(1)

	select {
	case <-exited:
	case <-time.After(time.Second):
		assert.Fail(t, "did not exit after being orphaned")
	}
}
