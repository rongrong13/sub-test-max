// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package trigger

import "sync"

type Cond struct {
	mu sync.Mutex
	ch chan struct{}
	on bool
}

func (c *Cond) Ready() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch == nil {
		c.ch = make(chan struct{})
	}
	return c.ch
}

func (c *Cond) Set() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.on {
		return
	}
	c.on = true
	if c.ch == nil {
		c.ch = make(chan struct{})
	}
	close(c.ch)
}

func (c *Cond) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.on {
		return
	}
	c.on = false
	c.ch = make(chan struct{})
}
