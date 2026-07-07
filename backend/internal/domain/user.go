package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// UserRole is the access level of a logged-in user (cashier terminal / ERP).
// Waiters are not Users — they have a separate Waiter document and authenticate
// via QR tokens, not username/password. See top-level CLAUDE.md "Auth model".
type UserRole string

const (
	RoleAdmin UserRole = "admin" // can do everything: kasa + ERP
)

// User is a credentialed operator. For MVP there is exactly one admin user
// created at seed time. Multi-user / role granularity comes later.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Username     string        `bson:"username" json:"username"`
	PasswordHash string        `bson:"passwordHash" json:"-"` // bcrypt
	Role         UserRole      `bson:"role" json:"role"`
	CreatedAt    time.Time     `bson:"createdAt" json:"createdAt"`
	LastLoginAt  *time.Time    `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
}
