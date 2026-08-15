package iprisk

import (
	"testing"
)

func TestEmojiFor(t *testing.T) {
	cases := map[int]string{0: "⚪", 10: "⚪", 15: "🟢", 30: "🟢", 35: "🟡", 50: "🟡", 60: "🟠", 70: "🟠", 85: "🔴", 90: "🔴", 95: "⚫", 100: "⚫"}
	for score, want := range cases {
		if got := emojiFor(score); got != want {
			t.Errorf("emojiFor(%d) = %q, want %q", score, got, want)
		}
	}
}

func TestHasAny(t *testing.T) {
	if !hasAny([]string{"hosting", "proxy"}, "proxy") {
		t.Error("hasAny should find proxy")
	}
	if hasAny([]string{"hosting"}, "proxy", "vpn", "tor") {
		t.Error("hasAny should not match absent keys")
	}
}

func TestFullString(t *testing.T) {
	r := &Result{RiskScore: 15, RiskPct: "15%", Emoji: "🟢", IPAttr: "机房", IPSource: "原生"}
	if got, want := r.FullString(), "【🟢 机房|原生】"; got != want {
		t.Errorf("FullString = %q, want %q", got, want)
	}
}
