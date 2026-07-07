package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Waiter has no password. Auth flow: admin selects waiter from a dropdown on
// the cashier terminal, backend issues a one-shot QRLoginToken, waiter scans
// it on phone, exchange creates a Session. See top-level CLAUDE.md "Auth model".
type Waiter struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Name         string        `bson:"name" json:"name"`
	Phone        string        `bson:"phone,omitempty" json:"phone,omitempty"`
	Active       bool          `bson:"active" json:"active"` // employed
}

// QRLoginToken is a one-shot token shown as a QR on the cashier terminal.
// On scan, the /auth/qr/exchange endpoint marks it used and creates a Session.
// TTL index on ExpiresAt cleans up unused tokens.
type QRLoginToken struct {
	ID           bson.ObjectID  `bson:"_id,omitempty"`
	RestaurantID bson.ObjectID  `bson:"restaurantId"`
	WaiterID     bson.ObjectID  `bson:"waiterId"`            // who this QR is for
	Token        string         `bson:"token"`               // random URL-safe string
	ExpiresAt    time.Time      `bson:"expiresAt"`           // QR validity, ~5 min
	UsedAt       *time.Time     `bson:"usedAt,omitempty"`
	IssuedBy     bson.ObjectID  `bson:"issuedBy,omitempty"`  // admin user id
}

// SessionKind discriminates admin vs waiter sessions in the unified store.
type SessionKind string

const (
	SessionAdmin  SessionKind = "admin"
	SessionWaiter SessionKind = "waiter"
)

// Session is a logged-in browser session, identified by an opaque token stored
// in an HTTP-only cookie. Admin sessions bump ExpiresAt on every request (idle
// timeout). Waiter sessions have a fixed ExpiresAt computed at issue time
// (rollover at 03:00 Europe/Istanbul). See internal/auth.
type Session struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	RestaurantID bson.ObjectID `bson:"restaurantId"`
	Kind         SessionKind   `bson:"kind"`
	SubjectID    bson.ObjectID `bson:"subjectId"` // User.ID for admin, Waiter.ID for waiter
	Token        string        `bson:"token"`     // random opaque, indexed unique
	ExpiresAt    time.Time     `bson:"expiresAt"`
	LastSeenAt   time.Time     `bson:"lastSeenAt"`
	CreatedAt    time.Time     `bson:"createdAt"`
}
