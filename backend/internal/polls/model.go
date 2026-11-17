package polls

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Option struct {
	ID    string `bson:"id" json:"id"`
	Label string `bson:"label" json:"label"`
	Count int64  `bson:"count" json:"count"`
}

type Poll struct {
	ID        bson.ObjectID `bson:"_id" json:"-"`
	Owner     bson.ObjectID `bson:"owner" json:"-"`
	Slug      string        `bson:"slug" json:"slug"`
	Question  string        `bson:"question" json:"question"`
	Options   []Option      `bson:"options" json:"options"`
	Closed    bool          `bson:"closed" json:"closed"`
	Revision  int64         `bson:"revision" json:"revision"`
	Total     int64         `bson:"total" json:"total"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
}

type View struct {
	Poll
	SelectedOptionID string `json:"selectedOptionId,omitempty"`
}

type Ballot struct {
	PollID    bson.ObjectID `bson:"pollId"`
	Voter     string        `bson:"voter"`
	OptionID  string        `bson:"optionId"`
	CreatedAt time.Time     `bson:"createdAt"`
}

type CreateInput struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}
