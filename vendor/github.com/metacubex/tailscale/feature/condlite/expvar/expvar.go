// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build ts_enable_debug || ts_enable_clientmetrics || ts_enable_usermetrics

// Package expvar contains type aliases for expvar types, to allow conditionally
// excluding the package from builds.
package expvar

import "expvar"

type Int = expvar.Int
