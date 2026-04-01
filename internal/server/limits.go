package server

import "github.com/stockyard-dev/stockyard-saltlick/internal/license"

// Limits holds the feature limits for the current license tier.
// All int limits: 0 means unlimited (Pro tier only).
type Limits struct {
	MaxFlags          int  // 0 = unlimited (Pro)
	MaxEvalsPerMonth  int  // 0 = unlimited (Pro)
	RetentionDays     int  // 7 free, 90 pro
	PercentRollout    bool // Pro only
	UserTargeting     bool // Pro only
	Environments      bool // Pro only
	WebhookOnChange   bool // Pro only
}

var freeLimits = Limits{
	MaxFlags:         10,
	MaxEvalsPerMonth: 0, // unlimited evals even on free — the hook
	RetentionDays:    7,
	PercentRollout:   false,
	UserTargeting:    false,
	Environments:     false,
	WebhookOnChange:  false,
}

var proLimits = Limits{
	MaxFlags:         0,
	MaxEvalsPerMonth: 0,
	RetentionDays:    90,
	PercentRollout:   true,
	UserTargeting:    true,
	Environments:     true,
	WebhookOnChange:  true,
}

// LimitsFor returns the appropriate Limits for the given license info.
// nil info = no key set = free tier.
func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

// LimitReached returns true if the current count meets or exceeds the limit.
// A limit of 0 is treated as unlimited.
func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
