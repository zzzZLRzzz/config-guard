package rules

import (
	"config-guard/internal/config"
	"strings"
)

type WhitelistMatcher struct {
	correctness []config.CorrectnessWhitelist
	diff        []config.DiffWhitelist
}

func NewWhitelistMatcher(wl config.Whitelist) *WhitelistMatcher {
	return &WhitelistMatcher{
		correctness: wl.Correctness,
		diff:        wl.Diff,
	}
}

func (w *WhitelistMatcher) IsCorrectnessWhitelisted(ruleID, instance string) bool {
	for _, entry := range w.correctness {
		if entry.Rule != ruleID {
			continue
		}
		if entry.Instance == "" || entry.Instance == instance {
			return true
		}
	}
	return false
}

func (w *WhitelistMatcher) IsDiffWhitelisted(key string, left, right string) (mode string, matched bool) {
	pair := sortedPair(left, right)
	for _, entry := range w.diff {
		if !strings.HasPrefix(key, entry.Key) {
			continue
		}
		if len(entry.Pair) < 2 {
			return entry.Mode, true
		}
		entryPair := sortedPair(entry.Pair[0], entry.Pair[1])
		if entryPair[0] == pair[0] && entryPair[1] == pair[1] {
			return entry.Mode, true
		}
	}
	return "", false
}

func sortedPair(a, b string) [2]string {
	if a <= b {
		return [2]string{a, b}
	}
	return [2]string{b, a}
}
