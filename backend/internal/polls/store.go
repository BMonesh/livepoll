package polls

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	client         *mongo.Client
	polls, ballots *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{db.Client(), db.Collection("polls"), db.Collection("ballots")}
}

func (s *Store) Index(ctx context.Context) error {
	_, err := s.polls.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "owner", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	if err != nil {
		return err
	}
	_, err = s.ballots.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "pollId", Value: 1}, {Key: "voter", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	return err
}

func (s *Store) Find(ctx context.Context, slug string) (Poll, error) {
	var poll Poll
	err := s.polls.FindOne(ctx, bson.M{"slug": slug}).Decode(&poll)
	return poll, err
}

func (s *Store) list(ctx context.Context, owner bson.ObjectID, before time.Time) ([]Poll, error) {
	filter := bson.M{"owner": owner}
	if !before.IsZero() {
		filter["createdAt"] = bson.M{"$lt": before}
	}
	cursor, err := s.polls.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50))
	if err != nil {
		return nil, err
	}
	polls := make([]Poll, 0)
	err = cursor.All(ctx, &polls)
	return polls, err
}

func (s *Store) selected(ctx context.Context, pollID bson.ObjectID, voter string) (string, error) {
	var ballot Ballot
	err := s.ballots.FindOne(ctx, bson.M{"pollId": pollID, "voter": voter}).Decode(&ballot)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	}
	return ballot.OptionID, err
}

func (s *Store) close(ctx context.Context, poll Poll) (Poll, error) {
	filter := bson.M{"_id": poll.ID, "owner": poll.Owner, "closed": false}
	update := bson.M{"$set": bson.M{"closed": true}, "$inc": bson.M{"revision": 1}}
	var closed Poll
	err := s.polls.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&closed)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return s.Find(ctx, poll.Slug)
	}
	return closed, err
}

type voteResult struct {
	View
	changed bool
}

func (s *Store) vote(ctx context.Context, slug, voter, choice string) (voteResult, error) {
	session, err := s.client.StartSession()
	if err != nil {
		return voteResult{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		poll, err := s.Find(tx, slug)
		if err != nil {
			return nil, err
		}
		existing, err := s.selected(tx, poll.ID, voter)
		if err != nil {
			return nil, err
		}
		if existing != "" {
			return voteResult{View{poll, existing}, false}, nil
		}
		if err := checkChoice(poll, choice); err != nil {
			return nil, err
		}

		filter := bson.M{"_id": poll.ID, "closed": false, "options.id": choice}
		update := bson.M{"$inc": bson.M{"options.$.count": 1, "total": 1, "revision": 1}}
		var updated Poll
		err = s.polls.FindOneAndUpdate(tx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
		if err != nil {
			return nil, err
		}
		_, err = s.ballots.InsertOne(tx, Ballot{poll.ID, voter, choice, time.Now()})
		if err != nil {
			return nil, err
		}
		return voteResult{View{updated, choice}, true}, nil
	})
	if err != nil {
		return voteResult{}, err
	}
	return result.(voteResult), nil
}
