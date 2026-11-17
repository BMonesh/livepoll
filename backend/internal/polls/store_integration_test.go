package polls

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestConcurrentVoteAndRetry(t *testing.T) {
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set TEST_MONGO_URI to a MongoDB replica set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database("livepoll_test_" + bson.NewObjectID().Hex())
	defer db.Drop(context.Background())
	store := NewStore(db)
	if err := store.Index(ctx); err != nil {
		t.Fatal(err)
	}

	owner := bson.NewObjectID()
	poll, err := store.create(ctx, owner, "Lunch?", []Option{{ID: "1", Label: "Dosa"}, {ID: "2", Label: "Idli"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.closeOwned(ctx, poll.Slug, bson.NewObjectID()); !errors.Is(err, errForbidden) {
		t.Fatalf("foreign creator closed poll or received wrong error: %v", err)
	}

	// Thirty separate voter cookies arrive together. Two of the requests are retries.
	var wg sync.WaitGroup
	failures := make(chan error, 32)
	for i := range 32 {
		wg.Go(func() {
			voter := strconv.Itoa(i % 30)
			result, err := store.vote(ctx, poll.Slug, voter, "1")
			if err != nil {
				failures <- err
				return
			}
			if result.SelectedOptionID != "1" {
				failures <- fmt.Errorf("unexpected recorded choice: %q", result.SelectedOptionID)
			}
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}

	closed, err := store.closeOwned(ctx, poll.Slug, owner)
	if err != nil {
		t.Fatal(err)
	}
	if !closed.Closed || closed.Total != 30 || closed.Revision != 31 || closed.Options[0].Count != 30 {
		t.Fatalf("unexpected final results: %+v", closed)
	}
	count, err := store.ballots.CountDocuments(ctx, bson.M{"pollId": poll.ID})
	if err != nil || count != 30 {
		t.Fatalf("stored ballots: %d, err: %v", count, err)
	}
	retry, err := store.vote(ctx, poll.Slug, "0", "2")
	if err != nil || retry.SelectedOptionID != "1" || retry.changed {
		t.Fatalf("unexpected retry: %+v, %v", retry, err)
	}
	if _, err := store.vote(ctx, poll.Slug, "new", "2"); !errors.Is(err, errClosed) {
		t.Fatalf("new vote after close should fail: %v", err)
	}
}
