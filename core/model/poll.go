package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// PollData data stored for a poll
type PollData struct {
	UserID        string     `json:"userid" bson:"userid" validate:"required"`
	UserName      string     `json:"username" bson:"username" validate:"required"`
	ToMembersList []ToMember `json:"to_members" bson:"to_members"` // nil or empty means everyone; non-empty means visible to those user ids
	Question      string     `json:"question" bson:"question" validate:"required"`
	Options       []string   `json:"options" bson:"options" validate:"required,min=2,dive,required"`
	GroupID       string     `json:"group_id,omitempty" bson:"group_id"`
	Pin           int        `json:"pin,omitempty" bson:"pin" validate:"min=0,max=9999"`
	MultiChoice   bool       `json:"multi_choice" bson:"multi_choice"`
	Repeat        bool       `json:"repeat" bson:"repeat"`
	ShowResults   bool       `json:"show_results" bson:"show_results"`
	Stadium       string     `json:"stadium" bson:"stadium"`
	Geo           bool       `json:"geo_fence" bson:"geo_fence"`
	Status        string     `json:"status" bson:"status" validate:"required,oneof=created started"`
	DateCreated   time.Time  `json:"date_created" bson:"date_created"`
	DateUpdated   time.Time  `json:"date_updated" bson:"date_updated"`
} // @name PollData

// UserHasAccess Checks if the user has read and write access to the poll object
func (pd *PollData) UserHasAccess(userID string) bool {

	if pd.UserID == userID {
		return true
	}

	if len(pd.ToMembersList) > 0 {
		for _, memberDef := range pd.ToMembersList {
			if memberDef.UserID == userID {
				return true
			}
		}
		return false
	}

	return true
}

// ToMember represents to(destination) member entity
type ToMember struct {
	UserID     string `json:"user_id" bson:"user_id"`
	ExternalID string `json:"external_id" bson:"external_id"`
	Name       string `json:"name" bson:"name"`
	Email      string `json:"email" bson:"email"`
} //@name ToMember

// Poll wraps the entire record
type Poll struct {
	PollData  `json:"poll" bson:"poll"`
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Responses []PollVote         `json:"responses" bson:"responses,omitempty" validate:"max=0"`
	Results   []int              `json:"results" bson:"results,omitempty" validate:"max=0"`
} // @name Poll

// ToPollResult converts to PollResult
func (poll *Poll) ToPollResult(currentUserID string) PollResult {

	result := PollResult{
		PollData: poll.PollData,
	}

	votes := make(map[int]bool)
	votersMap := make(map[string]bool)

	result.ID = poll.ID
	result.PollData = poll.PollData

	count := len(result.PollData.Options)
	result.Results = make([]int, count)

	if len(poll.Responses) > 0 {
		for _, e := range poll.Responses {
			votersMap[e.UserID] = true

			userVoted := poll.UserID == currentUserID

			for _, a := range e.Answer {
				if a >= 0 && a < count {
					if userVoted {
						votes[a] = true
					}
					result.Results[a]++
				}
			}
		}
	} else {
		copy(result.Results, poll.Results)
	}

	result.UniqueVotersCount = len(votersMap)

	for _, n := range result.Results {
		result.Total += n
	}

	if l := len(votes); l > 0 {
		result.Voted = make([]int, l)
		i := 0
		for k := range votes {
			result.Voted[i] = k
			i++
		}
	}

	return result
}

// PollVote data stored for each response
type PollVote struct {
	UserID  string    `json:"userid" validate:"required"`
	Answer  []int     `json:"answer" validate:"required,min=1"`
	Created time.Time `json:"created"`
} // @name PollVote

// PollResult wraps poll result
type PollResult struct {
	PollData          `json:"poll" bson:""`
	ID                primitive.ObjectID `json:"id"`
	Voted             []int              `json:"voted,omitempty"`
	Results           []int              `json:"results"`
	UniqueVotersCount int                `json:"unique_voters_count"`
	Total             int                `json:"total"`
} // @name PollResult
