/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package webutil

import (
	"testing"

	"github.com/blend/go-sdk/assert"
)

func TestLocalIP(t *testing.T) {
	assert := assert.New(t)

	assert.NotEmpty(LocalIP())
}
