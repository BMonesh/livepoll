package polls

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"livepoll/internal/auth"
)

var (
	errClosed    = errors.New("Voting is closed.")
	errChoice    = errors.New("Choose one of the available options.")
	errForbidden = errors.New("You can only manage your own polls.")
)

func validate(input CreateInput) (string, []Option, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" || utf8.RuneCountInString(question) > 160 {
		return "", nil, errors.New("Enter a question of 1 to 160 characters.")
	}
	if len(input.Options) < 2 || len(input.Options) > 6 {
		return "", nil, errors.New("Add between two and six options.")
	}
	labels := make([]Option, 0, len(input.Options))
	seen := make(map[string]bool, len(input.Options))
	for i, raw := range input.Options {
		label := strings.TrimSpace(raw)
		if label == "" || utf8.RuneCountInString(label) > 80 {
			return "", nil, errors.New("Each option must contain 1 to 80 characters.")
		}
		key := strings.ToLower(strings.Join(strings.Fields(label), " "))
		if seen[key] {
			return "", nil, errors.New("Each option must be different.")
		}
		seen[key] = true
		labels = append(labels, Option{ID: strconv.Itoa(i + 1), Label: label})
	}
	return question, labels, nil
}

func checkChoice(poll Poll, choice string) error {
	if poll.Closed {
		return errClosed
	}
	for _, option := range poll.Options {
		if option.ID == choice {
			return nil
		}
	}
	return errChoice
}

func (s *Store) create(ctx context.Context, owner bson.ObjectID, question string, options []Option) (Poll, error) {
	slug, err := auth.Token(16)
	if err != nil {
		return Poll{}, err
	}
	poll := Poll{ID: bson.NewObjectID(), Owner: owner, Slug: slug, Question: question, Options: options, CreatedAt: time.Now().UTC()}
	_, err = s.polls.InsertOne(ctx, poll)
	return poll, err
}

func (s *Store) closeOwned(ctx context.Context, slug string, owner bson.ObjectID) (Poll, error) {
	poll, err := s.Find(ctx, slug)
	if err != nil {
		return Poll{}, err
	}
	if poll.Owner != owner {
		return Poll{}, errForbidden
	}
	if poll.Closed {
		return poll, nil
	}
	return s.close(ctx, poll)
}
