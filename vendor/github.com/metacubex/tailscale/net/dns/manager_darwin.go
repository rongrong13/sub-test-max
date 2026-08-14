// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package dns

import (
	"github.com/metacubex/tailscale/control/controlknobs"
	"github.com/metacubex/tailscale/health"
	"github.com/metacubex/tailscale/types/logger"
	"github.com/metacubex/tailscale/util/eventbus"
	"github.com/metacubex/tailscale/util/syspolicy/policyclient"
)

// NewOSConfigurator creates a no-op OS configurator on macOS for embedded use.
// Mihomo owns system DNS configuration and consumes Tailscale DNS through its
// own DNS transport instead of letting tsnet write /etc/resolver files.
func NewOSConfigurator(logger.Logf, *health.Tracker, *eventbus.Bus, policyclient.Client, *controlknobs.Knobs, string) (OSConfigurator, error) {
	return NewNoopManager()
}
