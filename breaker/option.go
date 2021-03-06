/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package breaker

import (
	"time"
)

// Option is a mutator for a breaker.
type Option func(*Breaker) error

// OptHalfOpenMaxActions sets the HalfOpenMaxActions.
func OptHalfOpenMaxActions(maxActions int64) Option {
	return func(b *Breaker) error {
		b.HalfOpenMaxActions = maxActions
		return nil
	}
}

// OptClosedExpiryInterval sets the ClosedExpiryInterval.
func OptClosedExpiryInterval(interval time.Duration) Option {
	return func(b *Breaker) error {
		b.ClosedExpiryInterval = interval
		return nil
	}
}

// OptOpenExpiryInterval sets the OpenExpiryInterval.
func OptOpenExpiryInterval(interval time.Duration) Option {
	return func(b *Breaker) error {
		b.OpenExpiryInterval = interval
		return nil
	}
}

// OptConfig sets the breaker based on a config.
func OptConfig(cfg Config) Option {
	return func(b *Breaker) error {
		b.HalfOpenMaxActions = cfg.HalfOpenMaxActions
		b.ClosedExpiryInterval = cfg.ClosedExpiryInterval
		b.OpenExpiryInterval = cfg.OpenExpiryInterval
		return nil
	}
}

// OptOpenAction sets the open action on the breaker.
func OptOpenAction(action Action) Option {
	return func(b *Breaker) error {
		b.OpenAction = action
		return nil
	}
}

// OptOnStateChange sets the OnStateChange handler on the breaker.
func OptOnStateChange(handler OnStateChangeHandler) Option {
	return func(b *Breaker) error {
		b.OnStateChange = handler
		return nil
	}
}

// OptShouldOpenProvider sets the ShouldCloseProvider provider on the breaker.
func OptShouldOpenProvider(provider ShouldOpenProvider) Option {
	return func(b *Breaker) error {
		b.ShouldOpenProvider = provider
		return nil
	}
}

// OptNowProvider sets the now provider on the breaker.
func OptNowProvider(provider NowProvider) Option {
	return func(b *Breaker) error {
		b.NowProvider = provider
		return nil
	}
}
