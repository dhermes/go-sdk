/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package async

import (
	"context"
	"sync"
	"testing"

	"github.com/blend/go-sdk/assert"
)

func TestParallelQueue(t *testing.T) {
	assert := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(8)
	q := NewQueue(func(_ context.Context, obj interface{}) error {
		defer wg.Done()
		return nil
	})

	go func() { _ = q.Start() }()
	<-q.Latch.NotifyStarted()

	assert.True(q.Latch.IsStarted())

	for x := 0; x < 8; x++ {
		q.Enqueue("hello")
	}

	wg.Wait()
	q.Close()
	assert.False(q.Latch.IsStarted())
}
