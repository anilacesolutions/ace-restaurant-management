package auth

import "time"

// WaiterSessionRolloverHour is the local hour at which all waiter sessions
// expire — the "day boundary" for the restaurant. Hardcoded for MVP; the
// product plan calls for moving this to an admin-configurable restaurant
// setting later. See chat decision 2026-06-10.
const WaiterSessionRolloverHour = 3

// NextRollover returns the next instance of WaiterSessionRolloverHour:00 in
// the given location, strictly after `from`. So at 18:00 it returns the
// next-day 03:00; at 02:00 it returns today's 03:00 (one hour later).
func NextRollover(from time.Time, loc *time.Location) time.Time {
	t := from.In(loc)
	candidate := time.Date(t.Year(), t.Month(), t.Day(), WaiterSessionRolloverHour, 0, 0, 0, loc)
	if !candidate.After(t) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}
