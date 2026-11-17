package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalid     = errors.New("Enter a valid email and a password of 10 to 72 bytes.")
	errCredentials = errors.New("Email or password is incorrect.")
	errEmailTaken  = errors.New("An account already exists for this email.")
)

const lifetime = 7 * 24 * time.Hour

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Token(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func Fingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) signup(ctx context.Context, input Credentials) (User, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || len(input.Password) < 10 || len(input.Password) > 72 {
		return User{}, errInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	user := User{ID: bson.NewObjectID(), Email: email, PasswordHash: hash}
	_, err = s.users.InsertOne(ctx, user)
	if mongo.IsDuplicateKeyError(err) {
		return User{}, errEmailTaken
	}
	return user, err
}

func (s *Store) login(ctx context.Context, input Credentials) (User, error) {
	if len(input.Password) > 72 {
		return User{}, errCredentials
	}
	user, err := s.byEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Match the password verification cost for unknown accounts.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"), []byte(input.Password))
		return User{}, errCredentials
	}
	if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(input.Password)) != nil {
		return User{}, errCredentials
	}
	return user, nil
}

func (s *Store) startSession(ctx context.Context, user User, previous string) (string, error) {
	if previous != "" {
		if _, err := s.sessions.DeleteOne(ctx, bson.M{"_id": Fingerprint(previous)}); err != nil {
			return "", err
		}
	}
	token, err := Token(32)
	if err != nil {
		return "", err
	}
	_, err = s.sessions.InsertOne(ctx, session{Fingerprint(token), user.ID, time.Now().Add(lifetime)})
	return token, err
}
