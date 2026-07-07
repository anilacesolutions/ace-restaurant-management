package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrBadCredentials = errors.New("invalid credentials")
	ErrSessionExpired = errors.New("session expired")
	ErrSessionMissing = errors.New("no session")
	ErrTokenInvalid   = errors.New("invalid or used token")
	ErrWaiterNotFound = errors.New("waiter not found")
)

// AdminIdleTimeout is the inactivity window for admin sessions. Hardcoded for
// MVP; can later live on the restaurant settings.
const AdminIdleTimeout = 30 * time.Minute

// QRTokenTTL is how long a freshly issued QR remains scannable.
const QRTokenTTL = 5 * time.Minute

// Service owns auth-related collections (users, sessions, qrLoginTokens) and
// the in-restaurant timezone for waiter rollover math.
type Service struct {
	db           *mongo.Database
	restaurantID bson.ObjectID
	tz           *time.Location
}

func New(db *mongo.Database, restaurantID bson.ObjectID, tz *time.Location) *Service {
	return &Service{db: db, restaurantID: restaurantID, tz: tz}
}

func (s *Service) usersColl() *mongo.Collection    { return s.db.Collection("users") }
func (s *Service) sessionsColl() *mongo.Collection { return s.db.Collection("sessions") }
func (s *Service) qrColl() *mongo.Collection       { return s.db.Collection("qrLoginTokens") }
func (s *Service) waitersColl() *mongo.Collection  { return s.db.Collection("waiters") }

// ----- Admin: login / logout / me ----------------------------------------

// LoginAdmin verifies credentials, creates a fresh admin session, and returns
// it. lastLoginAt on the user is updated as a side effect.
func (s *Service) LoginAdmin(ctx context.Context, username, password string) (domain.Session, domain.User, error) {
	var u domain.User
	err := s.usersColl().FindOne(ctx, bson.M{
		"restaurantId": s.restaurantID,
		"username":     username,
	}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Session{}, domain.User{}, ErrBadCredentials
		}
		return domain.Session{}, domain.User{}, fmt.Errorf("find user: %w", err)
	}
	if u.Role != domain.RoleAdmin {
		return domain.Session{}, domain.User{}, ErrBadCredentials
	}
	if !VerifyPassword(u.PasswordHash, password) {
		return domain.Session{}, domain.User{}, ErrBadCredentials
	}

	now := time.Now().UTC()
	token, err := NewToken(32)
	if err != nil {
		return domain.Session{}, domain.User{}, fmt.Errorf("token: %w", err)
	}
	sess := domain.Session{
		ID:           bson.NewObjectID(),
		RestaurantID: s.restaurantID,
		Kind:         domain.SessionAdmin,
		SubjectID:    u.ID,
		Token:        token,
		ExpiresAt:    now.Add(AdminIdleTimeout),
		LastSeenAt:   now,
		CreatedAt:    now,
	}
	if _, err := s.sessionsColl().InsertOne(ctx, sess); err != nil {
		return domain.Session{}, domain.User{}, fmt.Errorf("insert session: %w", err)
	}

	_, _ = s.usersColl().UpdateOne(ctx,
		bson.M{"_id": u.ID},
		bson.M{"$set": bson.M{"lastLoginAt": now}},
	)
	u.LastLoginAt = &now
	return sess, u, nil
}

// LookupAdmin validates an admin session token; on success bumps lastSeenAt
// and expiresAt. Returns ErrSessionExpired when stale, ErrSessionMissing when
// not found.
func (s *Service) LookupAdmin(ctx context.Context, token string) (domain.Session, domain.User, error) {
	return s.lookup(ctx, token, domain.SessionAdmin)
}

// LookupWaiter validates a waiter session token. Does NOT extend expiry —
// waiter sessions have a fixed rollover.
func (s *Service) LookupWaiter(ctx context.Context, token string) (domain.Session, domain.Waiter, error) {
	sess, _, err := s.lookup(ctx, token, domain.SessionWaiter)
	if err != nil {
		return domain.Session{}, domain.Waiter{}, err
	}
	var w domain.Waiter
	if err := s.waitersColl().FindOne(ctx, bson.M{"_id": sess.SubjectID}).Decode(&w); err != nil {
		return domain.Session{}, domain.Waiter{}, fmt.Errorf("find waiter: %w", err)
	}
	return sess, w, nil
}

func (s *Service) lookup(ctx context.Context, token string, kind domain.SessionKind) (domain.Session, domain.User, error) {
	if token == "" {
		return domain.Session{}, domain.User{}, ErrSessionMissing
	}
	var sess domain.Session
	err := s.sessionsColl().FindOne(ctx, bson.M{
		"token":        token,
		"kind":         kind,
		"restaurantId": s.restaurantID,
	}).Decode(&sess)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Session{}, domain.User{}, ErrSessionMissing
		}
		return domain.Session{}, domain.User{}, fmt.Errorf("find session: %w", err)
	}

	now := time.Now().UTC()
	if !sess.ExpiresAt.After(now) {
		return domain.Session{}, domain.User{}, ErrSessionExpired
	}

	if kind == domain.SessionAdmin {
		newExp := now.Add(AdminIdleTimeout)
		_, _ = s.sessionsColl().UpdateOne(ctx,
			bson.M{"_id": sess.ID},
			bson.M{"$set": bson.M{"lastSeenAt": now, "expiresAt": newExp}},
		)
		sess.LastSeenAt = now
		sess.ExpiresAt = newExp

		var u domain.User
		if err := s.usersColl().FindOne(ctx, bson.M{"_id": sess.SubjectID}).Decode(&u); err != nil {
			return domain.Session{}, domain.User{}, fmt.Errorf("find user: %w", err)
		}
		return sess, u, nil
	}
	// Waiter sessions: just touch lastSeenAt without bumping expiresAt.
	_, _ = s.sessionsColl().UpdateOne(ctx,
		bson.M{"_id": sess.ID},
		bson.M{"$set": bson.M{"lastSeenAt": now}},
	)
	sess.LastSeenAt = now
	return sess, domain.User{}, nil
}

// Logout deletes a session by token. Idempotent.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.sessionsColl().DeleteOne(ctx, bson.M{"token": token})
	return err
}

// ----- QR: issue / exchange ----------------------------------------------

// IssueQR creates a one-shot QR login token for the given waiter. Returns the
// token string and its expiry; the caller embeds the token in the QR URL.
func (s *Service) IssueQR(ctx context.Context, waiterID, issuedBy bson.ObjectID) (string, time.Time, error) {
	var w domain.Waiter
	err := s.waitersColl().FindOne(ctx, bson.M{
		"_id":          waiterID,
		"restaurantId": s.restaurantID,
		"active":       true,
	}).Decode(&w)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", time.Time{}, ErrWaiterNotFound
		}
		return "", time.Time{}, fmt.Errorf("find waiter: %w", err)
	}

	token, err := NewToken(24)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(QRTokenTTL)
	doc := domain.QRLoginToken{
		ID:           bson.NewObjectID(),
		RestaurantID: s.restaurantID,
		WaiterID:     waiterID,
		Token:        token,
		ExpiresAt:    expires,
		IssuedBy:     issuedBy,
	}
	if _, err := s.qrColl().InsertOne(ctx, doc); err != nil {
		return "", time.Time{}, fmt.Errorf("insert qr token: %w", err)
	}
	return token, expires, nil
}

// ExchangeQR consumes a QR token and creates a waiter session expiring at the
// next 03:00 in the restaurant timezone.
func (s *Service) ExchangeQR(ctx context.Context, token string) (domain.Session, domain.Waiter, error) {
	now := time.Now().UTC()
	var qr domain.QRLoginToken
	err := s.qrColl().FindOneAndUpdate(ctx,
		bson.M{
			"token":        token,
			"restaurantId": s.restaurantID,
			"usedAt":       nil,
			"expiresAt":    bson.M{"$gt": now},
		},
		bson.M{"$set": bson.M{"usedAt": now}},
	).Decode(&qr)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Session{}, domain.Waiter{}, ErrTokenInvalid
		}
		return domain.Session{}, domain.Waiter{}, fmt.Errorf("consume qr: %w", err)
	}

	var w domain.Waiter
	if err := s.waitersColl().FindOne(ctx, bson.M{"_id": qr.WaiterID}).Decode(&w); err != nil {
		return domain.Session{}, domain.Waiter{}, fmt.Errorf("find waiter: %w", err)
	}

	sessTok, err := NewToken(32)
	if err != nil {
		return domain.Session{}, domain.Waiter{}, err
	}
	sess := domain.Session{
		ID:           bson.NewObjectID(),
		RestaurantID: s.restaurantID,
		Kind:         domain.SessionWaiter,
		SubjectID:    qr.WaiterID,
		Token:        sessTok,
		ExpiresAt:    NextRollover(now, s.tz),
		LastSeenAt:   now,
		CreatedAt:    now,
	}
	if _, err := s.sessionsColl().InsertOne(ctx, sess); err != nil {
		return domain.Session{}, domain.Waiter{}, fmt.Errorf("insert session: %w", err)
	}
	return sess, w, nil
}
