package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type User struct {
	ID           bson.ObjectID `bson:"_id" json:"-"`
	Email        string        `bson:"email" json:"email"`
	PasswordHash []byte        `bson:"passwordHash" json:"-"`
}

type session struct {
	ID      string        `bson:"_id"`
	UserID  bson.ObjectID `bson:"userId"`
	Expires time.Time     `bson:"expires"`
}

type Store struct{ users, sessions *mongo.Collection }

func NewStore(db *mongo.Database) *Store {
	return &Store{db.Collection("users"), db.Collection("sessions")}
}

func (s *Store) Index(ctx context.Context) error {
	_, err := s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	_, err = s.sessions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "expires", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

func (s *Store) byEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	return user, err
}

func (s *Store) sessionUser(ctx context.Context, id string) (User, error) {
	var record session
	if err := s.sessions.FindOne(ctx, bson.M{"_id": id, "expires": bson.M{"$gt": time.Now()}}).Decode(&record); err != nil {
		return User{}, err
	}
	var user User
	err := s.users.FindOne(ctx, bson.M{"_id": record.UserID}).Decode(&user)
	return user, err
}
