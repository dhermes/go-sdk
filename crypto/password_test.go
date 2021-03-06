/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package crypto

import (
	"testing"

	"github.com/blend/go-sdk/assert"
)

func Test_PasswordHashAndMatch(t *testing.T) {
	assert := assert.New(t)
	password := "some-test-password-12345"
	hashedPassword, err := HashPassword(password)
	assert.Nil(err)
	assert.NotEqual("", hashedPassword)
	assert.True(PasswordMatchesHash(password, hashedPassword))
	assert.False(PasswordMatchesHash("something-else", hashedPassword))
}
