// Copyright 2022 Board of Trustees of the University of Illinois.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PollsFilter Wraps all possible filters that could be used for retrieving polls
type PollsFilter struct {
	Pin     *int     `json:"pin"`
	PollIDs []string `json:"poll_ids,omitempty"`
	MyPolls *bool    `json:"my_polls,omitempty"`
	//GroupPolls     *bool    `json:"group_polls,omitempty"`
	GroupIDs       []string `json:"group_ids,omitempty"`
	RespondedPolls *bool    `json:"responded_polls,omitempty"`
	Statuses       []string `json:"statuses,omitempty"`
	Offset         *int64   `json:"offset,omitempty"`
	Limit          *int64   `json:"limit,omitempty"`
} // @name PollsFilter

// PollData data stored for a poll
type PollData struct {
	UserID        string    `json:"userid" bson:"userid" validate:"required"`
	UserName      string    `json:"username" bson:"username" validate:"required"`
	ToMembersList ToMembers `json:"to_members" bson:"to_members"` // nil or empty means everyone; non-empty means visible to those user ids
	Question      string    `json:"question" bson:"question" validate:"required"`
	Options       []string  `json:"options" bson:"options" validate:"required,min=2,dive,required"`
	GroupID       *string   `json:"group_id,omitempty" bson:"group_id"`
	Pin           int       `json:"pin,omitempty" bson:"pin" validate:"min=0,max=9999"`
	MultiChoice   bool      `json:"multi_choice" bson:"multi_choice"`
	Repeat        bool      `json:"repeat" bson:"repeat"`
	ShowResults   bool      `json:"show_results" bson:"show_results"`
	Stadium       string    `json:"stadium" bson:"stadium"`
	Geo           bool      `json:"geo_fence" bson:"geo_fence"`
	Status        string    `json:"status" bson:"status" validate:"required,oneof=created started"`
	DateCreated   time.Time `json:"date_created" bson:"date_created"`
	DateUpdated   time.Time `json:"date_updated" bson:"date_updated"`
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

// ToMembers wrapper for list of to members
type ToMembers []ToMember // @name ToMembers

// ToNotificationRecipients converts to notification recipients
func (t ToMembers) ToNotificationRecipients() []UserRef {
	var recipients []UserRef
	for _, toMember := range t {
		recipients = append(recipients, UserRef{
			UserID: toMember.UserID,
			Name:   toMember.Name,
		})
	}
	return recipients
}

// ToMember represents to(destination) member entity
type ToMember struct {
	UserID     string `json:"user_id" bson:"user_id"`
	ExternalID string `json:"external_id" bson:"external_id"`
	Name       string `json:"name" bson:"name"`
	Email      string `json:"email" bson:"email"`
} //@name ToMember

// PollNotification wraps the entire record
type PollNotification struct {
	PollData  `json:"poll" bson:"poll"`
	OrgID     string             `json:"org_id" bson:"org_id"`
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	Responses []PollVote         `json:"responses" bson:"responses,omitempty" validate:"max=0"`
	Results   []int              `json:"results" bson:"results,omitempty" validate:"max=0"`
} // @name PollNotification

// ToPollResult converts to PollResult
func (poll *PollNotification) ToPollResult(currentUserID string) PollResult {

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

// Poll wraps the entire record
type Poll struct {
	PollData  `json:"poll" bson:"poll"`
	OrgID     string             `json:"org_id" bson:"org_id"`
	ID        primitive.ObjectID `json:"id" bson:"_id"`
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

			userVoted := e.UserID == currentUserID

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

// GetPollNotificationRecipients gets poll to members as notification recipients
func (poll *Poll) GetPollNotificationRecipients(currentUserID string) []UserRef {
	var recipients []UserRef
	if len(poll.ToMembersList) > 0 {
		for _, toMember := range poll.ToMembersList {
			if currentUserID != toMember.UserID {
				recipients = append(recipients, UserRef{
					UserID: toMember.UserID,
					Name:   toMember.Name,
				})
			}
		}
	}
	return recipients
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
