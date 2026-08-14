// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package rand backports the subset of math/rand/v2 used by Tailscale.
package rand

import (
	mrand "math/rand"
	"time"

	"golang.org/x/exp/constraints"
)

type PCG struct {
	r *mrand.Rand
}

func (p *PCG) Seed(seed1, seed2 uint64) {
	p.r = mrand.New(mrand.NewSource(int64(seed1 ^ (seed2 << 1))))
}

func (p *PCG) Uint64() uint64 {
	if p.r == nil {
		p.Seed(Uint64(), Uint64())
	}
	return uint64(p.r.Uint32())<<32 | uint64(p.r.Uint32())
}

func init() {
	mrand.Seed(time.Now().UnixNano())
}

func Uint64() uint64 {
	return uint64(mrand.Uint32())<<32 | uint64(mrand.Uint32())
}

func Float64() float64 {
	return mrand.Float64()
}

func IntN(n int) int {
	return mrand.Intn(n)
}

func N[T constraints.Integer](n T) T {
	if n <= 0 {
		panic("invalid argument to N")
	}
	return T(mrand.Int63n(int64(n)))
}

func Shuffle(n int, swap func(i, j int)) {
	mrand.Shuffle(n, swap)
}
