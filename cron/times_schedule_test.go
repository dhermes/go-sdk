/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package cron

import (
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestTimesSchedule(t *testing.T) {
	assert := assert.New(t)

	ts := time.Date(2020, 01, 23, 12, 11, 10, 9, time.UTC)
	schedule := Times(5, Every(time.Millisecond))
	assert.Equal(ts.Add(time.Millisecond), schedule.Next(ts))
	assert.Equal(ts.Add(time.Millisecond), schedule.Next(ts))
	assert.Equal(ts.Add(time.Millisecond), schedule.Next(ts))
	assert.Equal(ts.Add(time.Millisecond), schedule.Next(ts))
	assert.Equal(ts.Add(time.Millisecond), schedule.Next(ts))
	assert.True(schedule.Next(ts).IsZero())
}
