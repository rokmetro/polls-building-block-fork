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

package storage

import (
	"fmt"
	"log"
	"polls/core/model"
	"polls/driven/groups"
	"strconv"
	"time"

	"github.com/rokwire/rokwire-building-block-sdk-go/utils/errors"
	"github.com/rokwire/rokwire-building-block-sdk-go/utils/logging/logs"
	"github.com/rokwire/rokwire-building-block-sdk-go/utils/logging/logutils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// PollStatusCreated status created
	PollStatusCreated = "created"

	// PollStatusStarted status started
	PollStatusStarted = "started"

	// PollStatusTerminated status terminated
	PollStatusTerminated = "terminated"

	settingsKey   = "stadium"
	eventInterval = 100 * time.Millisecond
)

// Adapter implements the Storage interface
type Adapter struct {
	db     *database
	config *model.Config
}

// Start starts the storage
func (sa *Adapter) Start() error {
	err := sa.db.start()
	if err != nil {
		return err
	}

	err = sa.applyMultiTenancy()
	return err
}

// NewStorageAdapter creates a new storage adapter instance
func NewStorageAdapter(config *model.Config, logger *logs.Logger) *Adapter {
	timeout, err := strconv.Atoi(config.MongoTimeout)
	if err != nil {
		log.Println("Set default timeout - 500")
		timeout = 500
	}
	timeoutMS := time.Millisecond * time.Duration(timeout)

	db := &database{mongoDBAuth: config.MongoDBAuth, mongoDBName: config.MongoDBName, mongoTimeout: timeoutMS, logger: logger}
	return &Adapter{db: db, config: config}
}

// GetPolls retrieves all polls with an ability to filter
func (sa *Adapter) GetPolls(user *model.User, filter model.PollsFilter, filterByToMembers bool, membership *groups.GroupMembership) ([]model.Poll, error) {
	mongoFilter := bson.D{}
	if user != nil {
		mongoFilter = append(mongoFilter, primitive.E{Key: "org_id", Value: user.Claims.OrgID})
	}

	if len(filter.PollIDs) > 0 {
		reconstructedIDs := []primitive.ObjectID{}
		for _, id := range filter.PollIDs {
			if objID, err := primitive.ObjectIDFromHex(id); err == nil {
				reconstructedIDs = append(reconstructedIDs, objID)
			}
		}
		mongoFilter = append(mongoFilter, primitive.E{Key: "_id", Value: bson.M{"$in": reconstructedIDs}})
	}

	if filter.MyPolls != nil && *filter.MyPolls == true && filter.RespondedPolls != nil && *filter.RespondedPolls == true {
		mongoFilter = append(mongoFilter, primitive.E{Key: "$or", Value: []primitive.M{
			{"poll.userid": user.Claims.Subject},
			{"responses.userid": user.Claims.Subject},
		}})
	} else {
		if filter.MyPolls != nil && *filter.MyPolls == true {
			mongoFilter = append(mongoFilter, primitive.E{Key: "poll.userid", Value: user.Claims.Subject})
		}

		if filter.RespondedPolls != nil && *filter.RespondedPolls == true {
			mongoFilter = append(mongoFilter, primitive.E{Key: "responses.userid", Value: user.Claims.Subject})
		}
	}

	if filter.Pin != nil {
		mongoFilter = append(mongoFilter, primitive.E{Key: "poll.pin", Value: *filter.Pin})
	}

	if len(filter.GroupIDs) > 0 {
		mongoFilter = append(mongoFilter, primitive.E{Key: "poll.group_id", Value: bson.M{"$in": filter.GroupIDs}})
	}

	if len(filter.Statuses) > 0 {
		mongoFilter = append(mongoFilter, primitive.E{Key: "poll.status", Value: bson.M{"$in": filter.Statuses}})
	}

	if filterByToMembers {
		var innerFilter primitive.M
		if membership != nil && len(membership.GroupIDsAsAdmin) > 0 {
			innerFilter = primitive.M{"$or": []primitive.M{
				{"poll.group_id": bson.M{"$in": membership.GroupIDsAsAdmin}},
				{"poll.to_members.user_id": user.Claims.Subject},
			}}
		} else {
			innerFilter = primitive.M{"poll.to_members.user_id": user.Claims.Subject}
		}

		mongoFilter = append(mongoFilter, primitive.E{Key: "$or", Value: []primitive.M{
			{"poll.to_members": primitive.Null{}},
			{"poll.to_members": primitive.M{"$exists": true, "$size": 0}},
			{"poll.user_id": user.Claims.Subject},
			innerFilter,
		}})
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "poll.status", Value: 1}, {Key: "_id", Value: -1}})

	if filter.Limit != nil {
		findOptions.SetLimit(*filter.Limit)
	}
	if filter.Offset != nil {
		findOptions.SetSkip(*filter.Offset)
	}

	var list []model.Poll
	err := sa.db.polls.Find(mongoFilter, &list, findOptions)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// DeletePollsWithIDs Deletes polls
func (sa Adapter) DeletePollsWithIDs(orgID string, accountsIDs []string) error {
	filter := bson.D{
		primitive.E{Key: "org_id", Value: orgID},
		primitive.E{Key: "poll.userid", Value: bson.M{"$in": accountsIDs}},
	}

	_, err := sa.db.polls.DeleteMany(filter, nil)
	if err != nil {
		return errors.WrapErrorAction(logutils.ActionDelete, "user", nil, err)
	}
	return nil
}

// GetPoll retrieves a single poll
func (sa *Adapter) GetPoll(user *model.User, id string, filterByToMembers bool, membership *groups.GroupMembership) (*model.Poll, error) {

	if objID, err := primitive.ObjectIDFromHex(id); err == nil {
		filter := bson.D{
			primitive.E{Key: "org_id", Value: user.Claims.OrgID},
			primitive.E{Key: "_id", Value: objID},
		}

		if filterByToMembers {
			var innerFilter primitive.M
			if membership != nil && len(membership.GroupIDsAsAdmin) > 0 {
				innerFilter = primitive.M{"$or": []primitive.M{
					{"poll.group_id": bson.M{"$in": membership.GroupIDsAsAdmin}},
					{"poll.to_members.user_id": user.Claims.Subject},
				}}
			} else {
				innerFilter = primitive.M{"poll.to_members.user_id": user.Claims.Subject}
			}

			filter = append(filter, primitive.E{Key: "$or", Value: []primitive.M{
				{"poll.to_members": primitive.Null{}},
				{"poll.to_members": primitive.M{"$exists": true, "$size": 0}},
				{"poll.user_id": user.Claims.Subject},
				innerFilter,
			}})
		}

		var poll model.Poll
		err := sa.db.polls.FindOne(filter, &poll, &options.FindOneOptions{})
		if err != nil {
			fmt.Printf("error storage.Adapter.GetPoll(%s) - %s", id, err)
			return nil, fmt.Errorf("error storage.Adapter.GetPoll(%s) - %s", id, err)
		}

		return &poll, nil
	}

	fmt.Printf("error storage.Adapter.GetPoll(%s) - unable to construct obj id", id)
	return nil, fmt.Errorf("error storage.Adapter.GetPoll(%s) - unable to construct obj id", id)
}

// CreatePoll creates a poll
func (sa *Adapter) CreatePoll(user *model.User, poll model.Poll) (*model.Poll, error) {
	now := time.Now()
	poll.OrgID = user.Claims.OrgID
	poll.ID = primitive.NewObjectID()
	poll.UserID = user.Claims.Subject
	poll.UserName = user.Claims.Name
	poll.DateCreated = now
	poll.DateUpdated = &now

	_, err := sa.db.polls.InsertOne(poll)
	if err != nil {
		fmt.Printf("error storage.Adapter.CreatePoll(%s) - %s", poll.ID, err)
		return nil, fmt.Errorf("error storage.Adapter.CreatePoll(%s) - %s", poll.ID, err)
	}

	return &poll, nil
}

// UpdatePoll updates a poll
func (sa *Adapter) UpdatePoll(user *model.User, poll model.Poll) (*model.Poll, error) {

	if len(poll.ID) > 0 {

		now := time.Now().UTC()
		poll.DateUpdated = &now
		filter := bson.D{
			primitive.E{Key: "org_id", Value: user.Claims.OrgID},
			primitive.E{Key: "_id", Value: poll.ID},
		}

		update := bson.D{
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "poll.date_updated", Value: poll.DateUpdated},
				primitive.E{Key: "poll.to_members", Value: poll.ToMembersList},
				primitive.E{Key: "poll.pin", Value: poll.Pin},
				primitive.E{Key: "poll.question", Value: poll.Question},
				primitive.E{Key: "poll.options", Value: poll.Options},
				primitive.E{Key: "poll.group_id", Value: poll.GroupID},
				primitive.E{Key: "poll.multi_choice", Value: poll.MultiChoice},
				primitive.E{Key: "poll.repeat", Value: poll.Repeat},
				primitive.E{Key: "poll.show_results", Value: poll.ShowResults},
				primitive.E{Key: "poll.stadium", Value: poll.Stadium},
				primitive.E{Key: "poll.geo_fence", Value: poll.Geo},
				primitive.E{Key: "poll.status", Value: poll.Status},
			}},
		}

		_, err := sa.db.polls.UpdateOne(filter, update, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.UpdatePoll(%s) - %s", poll.ID, err)
			return nil, fmt.Errorf("error storage.Adapter.UpdatePoll(%s) - %s", poll.ID, err)
		}
	}

	return &poll, nil
}

// StartPoll starts an existing poll
func (sa *Adapter) StartPoll(user *model.User, pollID string) error {

	poll, err := sa.GetPoll(user, pollID, true, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.StartPoll(%s) - %s", pollID, err)
		return fmt.Errorf("error storage.Adapter.StartPoll(%s) - %s", pollID, err)
	}

	if poll != nil {
		if poll.Status != PollStatusStarted {
			poll.Status = PollStatusStarted
			_, err = sa.UpdatePoll(user, *poll)
			if err != nil {
				fmt.Printf("error storage.Adapter.StartPoll(%s) - %s", pollID, err)
				return fmt.Errorf("error storage.Adapter.StartPoll(%s) - %s", pollID, err)
			}
		}
	} else {
		return fmt.Errorf("error storage.Adapter.EndPoll(%s) - poll not found: %s", pollID, err)
	}

	return nil
}

// EndPoll ends an existing poll
func (sa *Adapter) EndPoll(user *model.User, pollID string) error {
	poll, err := sa.GetPoll(user, pollID, true, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.EndPoll(%s) - %s", pollID, err)
		return fmt.Errorf("error storage.Adapter.EndPoll(%s) - %s", pollID, err)
	}

	if poll != nil {
		if poll.Status != PollStatusTerminated {
			poll.Status = PollStatusTerminated
			_, err = sa.UpdatePoll(user, *poll)
			if err != nil {
				fmt.Printf("error storage.Adapter.EndPoll(%s) - %s", pollID, err)
				return fmt.Errorf("error storage.Adapter.EndPoll(%s) - %s", pollID, err)
			}
		}
	} else {
		return fmt.Errorf("error storage.Adapter.EndPoll(%s) - poll not found: %s", pollID, err)
	}

	return nil
}

// DeletePoll deletes a poll
func (sa *Adapter) DeletePoll(user *model.User, id string) error {
	if objID, err := primitive.ObjectIDFromHex(id); err == nil {
		filter := bson.D{
			primitive.E{Key: "org_id", Value: user.Claims.OrgID},
			primitive.E{Key: "_id", Value: objID},
		}
		_, err := sa.db.polls.DeleteOne(filter, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.DeletePoll(): error while delete poll (%s) - %s", id, err)
			return fmt.Errorf("error storage.Adapter.DeletePoll(): error while delete poll (%s) - %s", id, err)
		}

	}
	return nil

}

// VotePoll votes a poll
func (sa *Adapter) VotePoll(user *model.User, pollID string, vote model.PollVote) error {

	if objID, err := primitive.ObjectIDFromHex(pollID); err == nil {
		now := time.Now().UTC()
		vote.Created = now

		filter := bson.D{
			primitive.E{Key: "_id", Value: objID},
		}

		update := bson.D{
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "poll.date_updated", Value: now},
			}},
			primitive.E{Key: "$push", Value: bson.D{
				primitive.E{Key: "responses", Value: vote},
			}},
		}

		_, err := sa.db.polls.UpdateOne(filter, update, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.VotePoll(%s) - %s", pollID, err)
			return fmt.Errorf("error storage.Adapter.VotePoll(%s) - %s", pollID, err)
		}
	}
	return nil
}

// SetListener sets the upper layer listener for sending collection changed callbacks
func (sa *Adapter) SetListener(listener CollectionListener) {
	sa.db.listener = listener
}

// Event

func (m *database) onDataChanged(changeDoc map[string]interface{}) {
	if changeDoc == nil {
		return
	}
	log.Printf("onDataChanged: %+v\n", changeDoc)
	ns := changeDoc["ns"]
	if ns == nil {
		return
	}
	nsMap := ns.(map[string]interface{})
	coll := nsMap["coll"]

	record := changeDoc["fullDocument"]
	var recordMap map[string]interface{}
	if record != nil {
		recordMap = record.(map[string]interface{})
	}

	if m.listener != nil {
		m.listener.OnCollectionUpdated(coll.(string), recordMap)
	}
}

// GetSurvey retrieves a single survey
func (sa *Adapter) GetSurvey(user *model.User, id string) (*model.Survey, error) {
	filter := bson.M{"_id": id, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	var entry model.Survey
	err := sa.db.surveys.FindOne(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetSurvey(%s) - %s", id, err)
		return nil, fmt.Errorf("error storage.Adapter.GetSurvey(%s) - %s", id, err)
	}

	return &entry, nil
}

// CreateSurvey creates a poll
func (sa *Adapter) CreateSurvey(survey model.Survey) (*model.Survey, error) {
	_, err := sa.db.surveys.InsertOne(survey)
	if err != nil {
		fmt.Printf("error storage.Adapter.CreateSurvey(%s) - %s", survey.ID, err)
		return nil, fmt.Errorf("error storage.Adapter.CreateSurvey(%s) - %s", survey.ID, err)
	}

	return &survey, nil
}

// UpdateSurvey updates a survey
func (sa *Adapter) UpdateSurvey(user *model.User, survey model.Survey, admin bool) error {
	if len(survey.ID) > 0 {
		now := time.Now().UTC()
		filter := bson.M{"_id": survey.ID, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
		if !admin {
			filter["creator_id"] = user.Claims.Subject
		}
		update := bson.M{"$set": bson.M{
			"title":                 survey.Title,
			"more_info":             survey.MoreInfo,
			"data":                  survey.Data,
			"scored":                survey.Scored,
			"result_rules":          survey.ResultRules,
			"type":                  survey.Type,
			"stats":                 survey.SurveyStats,
			"sensitive":             survey.Sensitive,
			"default_data_key":      survey.DefaultDataKey,
			"default_data_key_rule": survey.DefaultDataKeyRule,
			"constants":             survey.Constants,
			"strings":               survey.Strings,
			"sub_rules":             survey.SubRules,
			"date_updated":          now,
		}}

		res, err := sa.db.surveys.UpdateOne(filter, update, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.UpdateSurvey(%s) - %s", survey.ID, err)
			return fmt.Errorf("error storage.Adapter.UpdateSurvey(%s) - %s", survey.ID, err)
		}
		if res.ModifiedCount != 1 {
			fmt.Printf("storage.Adapter.UpdateSurvey(%s) invalid id", survey.ID)
			return fmt.Errorf("storage.Adapter.UpdateSurvey(%s) invalid id", survey.ID)
		}
	}

	return nil
}

// DeleteSurvey deletes a survey
func (sa *Adapter) DeleteSurvey(user *model.User, id string, admin bool) error {
	filter := bson.M{"_id": id, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	if !admin {
		filter["creator_id"] = user.Claims.Subject
	}
	res, err := sa.db.surveys.DeleteOne(filter, nil)
	if err != nil {
		return fmt.Errorf("error storage.Adapter.DeleteSurvey(): error while delete survey (%s) - %s", id, err)
	}
	if res.DeletedCount != 1 {
		fmt.Printf("storage.Adapter.DeleteSurvey(%s) invalid id", id)
		return fmt.Errorf("storage.Adapter.DeleteSurvey(%s) invalid id", id)
	}

	return nil
}

// GetSurveyResponse gets a survey response by ID
func (sa *Adapter) GetSurveyResponse(user *model.User, id string) (*model.SurveyResponse, error) {
	filter := bson.M{"_id": id, "user_id": user.Claims.Subject, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	var entry model.SurveyResponse
	err := sa.db.surveyResponses.FindOne(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetSurveyResponse(%s) - %s", id, err)
		return nil, fmt.Errorf("error storage.Adapter.GetSurveyResponse(%s) - %s", id, err)
	}
	return &entry, nil
}

// GetSurveyResponseByUserID gets a survey response by user ID
func (sa *Adapter) GetSurveyResponseByUserID(user *model.User) ([]model.SurveyResponse, error) {
	filter := bson.M{"user_id": user.Claims.Subject}
	var entry []model.SurveyResponse
	err := sa.db.surveyResponses.Find(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetSurveyResponseByUserID - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetSurveyResponseByUserID - %s", err)
	}
	return entry, nil
}

// GetSurveysByUserID gets a surveys by user ID
func (sa *Adapter) GetSurveysByUserID(user *model.User) ([]model.Survey, error) {
	filter := bson.M{"creator_id": user.Claims.Subject}
	var entry []model.Survey
	err := sa.db.surveys.Find(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetSurveysByUserID - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetSurveysByUserID - %s", err)
	}
	return entry, nil
}

// GetSurveyResponses gets matching surveys for a user
func (sa *Adapter) GetSurveyResponses(user *model.User, surveyIDs []string, surveyTypes []string, startDate *time.Time, endDate *time.Time, limit *int, offset *int) ([]model.SurveyResponse, error) {
	filter := bson.M{"user_id": user.Claims.Subject, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	if len(surveyIDs) > 0 {
		filter["survey._id"] = bson.M{"$in": surveyIDs}
	}
	if len(surveyTypes) > 0 {
		filter["survey.type"] = bson.M{"$in": surveyTypes}
	}
	if startDate != nil || endDate != nil {
		dateFilter := bson.M{}
		if startDate != nil {
			dateFilter["$gte"] = startDate
		}
		if endDate != nil {
			dateFilter["$lt"] = endDate
		}
		filter["date_created"] = dateFilter
	}

	opts := options.Find().SetSort(bson.M{"date_created": -1})
	if limit != nil {
		opts.SetLimit(int64(*limit))
	}
	if offset != nil {
		opts.SetSkip(int64(*offset))
	}
	var results []model.SurveyResponse
	err := sa.db.surveyResponses.Find(filter, &results, opts)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetSurveyResponses - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetSurveyResponses - %s", err)
	}
	return results, nil
}

// CreateSurveyResponse creates a new survey response
func (sa *Adapter) CreateSurveyResponse(surveyResponse model.SurveyResponse) (*model.SurveyResponse, error) {
	_, err := sa.db.surveyResponses.InsertOne(surveyResponse)
	if err != nil {
		fmt.Printf("error storage.Adapter.CreateSurveyResponse(%s) - %s", surveyResponse.ID, err)
		return nil, fmt.Errorf("error storage.Adapter.CreateSurveyResponse(%s) - %s", surveyResponse.ID, err)
	}
	return &surveyResponse, nil
}

// UpdateSurveyResponse updates an existing service response
func (sa *Adapter) UpdateSurveyResponse(user *model.User, id string, survey model.Survey) error {
	if len(id) > 0 {
		now := time.Now().UTC()
		filter := bson.M{"_id": id, "user_id": user.Claims.Subject, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
		update := bson.M{"$set": bson.M{
			"survey":       survey,
			"date_updated": now,
		}}

		res, err := sa.db.surveyResponses.UpdateOne(filter, update, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.UpdateSurveyResponse(%s) - %s", id, err)
			return fmt.Errorf("error storage.Adapter.UpdateSurveyResponse(%s) - %s", id, err)
		}
		if res.ModifiedCount != 1 {
			fmt.Printf("storage.Adapter.UpdateSurveyResponse(%s) invalid id", id)
			return fmt.Errorf("storage.Adapter.UpdateSurveyResponse(%s) invalid id", id)
		}
	}
	return nil
}

// DeleteSurveyResponse deletes a survey response
func (sa *Adapter) DeleteSurveyResponse(user *model.User, id string) error {
	filter := bson.M{"_id": id, "user_id": user.Claims.Subject, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	res, err := sa.db.surveyResponses.DeleteOne(filter, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.DeleteSurveyResponse(%s) - %s", id, err)
		return fmt.Errorf("error storage.Adapter.DeleteSurveyResponse(): error while delete survey response (%s) - %s", id, err)
	}
	if res.DeletedCount != 1 {
		fmt.Printf("storage.Adapter.DeleteSurveyResponse(%s) invalid id", id)
		return fmt.Errorf("storage.Adapter.DeleteSurveyResponse(%s) invalid id", id)
	}
	return nil
}

// DeleteSurveyResponses deletes matching surveys
func (sa *Adapter) DeleteSurveyResponses(user *model.User, surveyIDs []string, surveyTypes []string, startDate *time.Time, endDate *time.Time) error {
	filter := bson.M{"user_id": user.Claims.Subject, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	if len(surveyIDs) > 0 {
		filter["survey._id"] = bson.M{"$in": surveyIDs}
	}
	if len(surveyTypes) > 0 {
		filter["survey.type"] = bson.M{"$in": surveyTypes}
	}
	if startDate != nil || endDate != nil {
		dateFilter := bson.M{}
		if startDate != nil {
			dateFilter["$gte"] = startDate
		}
		if endDate != nil {
			dateFilter["$lt"] = endDate
		}
		filter["date_created"] = dateFilter
	}

	result, err := sa.db.surveyResponses.DeleteMany(filter, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.DeleteSurveyResponses - %s", err)
		return fmt.Errorf("error storage.Adapter.DeleteSurveyResponses - %s", err)
	}
	if result.DeletedCount == 0 {
		fmt.Printf("storage.Adapter.DeleteSurveyResponses: No deleted survey responses")
	}
	return nil
}

// DeleteSurveyResponsesWithIDs Deletes survey responses
func (sa Adapter) DeleteSurveyResponsesWithIDs(appID string, orgID string, accountsIDs []string) error {
	filter := bson.D{
		primitive.E{Key: "app_id", Value: appID},
		primitive.E{Key: "org_id", Value: orgID},
		primitive.E{Key: "user_id", Value: bson.M{"$in": accountsIDs}},
	}

	_, err := sa.db.surveyResponses.DeleteMany(filter, nil)
	if err != nil {
		return errors.WrapErrorAction(logutils.ActionDelete, "user", nil, err)
	}
	return nil
}

// DeleteSurveysWithIDs Deletes surveys
func (sa Adapter) DeleteSurveysWithIDs(appID string, orgID string, accountsIDs []string) error {
	filter := bson.D{
		primitive.E{Key: "app_id", Value: appID},
		primitive.E{Key: "org_id", Value: orgID},
		primitive.E{Key: "creator_id", Value: bson.M{"$in": accountsIDs}},
	}

	_, err := sa.db.surveys.DeleteMany(filter, nil)
	if err != nil {
		return errors.WrapErrorAction(logutils.ActionDelete, "user", nil, err)
	}
	return nil
}

// GetAlertContacts retrieves all alert contacts
func (sa *Adapter) GetAlertContacts(user *model.User) ([]model.AlertContact, error) {
	filter := bson.M{"org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	var entry []model.AlertContact
	err := sa.db.alertContacts.Find(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetAlertContacts - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetAlertContacts - %s", err)
	}

	return entry, nil
}

// GetAlertContact retrieves a single alert contact
func (sa *Adapter) GetAlertContact(user *model.User, id string) (*model.AlertContact, error) {
	filter := bson.M{"_id": id, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	var entry model.AlertContact
	err := sa.db.alertContacts.FindOne(filter, &entry, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetAlertContact(%s) - %s", id, err)
		return nil, fmt.Errorf("error storage.Adapter.GetAlertContact(%s) - %s", id, err)
	}

	return &entry, nil
}

// GetAlertContactsByKey gets all alert contacts that share the key in the filter
func (sa *Adapter) GetAlertContactsByKey(key string, user *model.User) ([]model.AlertContact, error) {
	filter := bson.M{"key": key, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	var results []model.AlertContact
	err := sa.db.alertContacts.Find(filter, &results, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetAlertContactsByKey - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetAlertContactsByKey - %s", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("error on Application.createSurveyAlert: No contacts found with key %s", key)
	}

	return results, nil
}

// CreateAlertContact creates an alert contact
func (sa *Adapter) CreateAlertContact(alertContact model.AlertContact) (*model.AlertContact, error) {
	_, err := sa.db.alertContacts.InsertOne(alertContact)
	if err != nil {
		fmt.Printf("error storage.Adapter.CreateAlertContact(%s) - %s", alertContact.ID, err)
		return nil, fmt.Errorf("error storage.Adapter.CreateAlertContact(%s) - %s", alertContact.ID, err)
	}

	return &alertContact, nil
}

// UpdateAlertContact updates an alert contact
func (sa *Adapter) UpdateAlertContact(user *model.User, id string, alertContact model.AlertContact) error {
	if len(id) > 0 {
		now := time.Now().UTC()
		filter := bson.M{"_id": id, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
		update := bson.M{"$set": bson.M{
			"key":          alertContact.Key,
			"type":         alertContact.Type,
			"address":      alertContact.Address,
			"params":       alertContact.Params,
			"date_updated": now,
		}}

		res, err := sa.db.alertContacts.UpdateOne(filter, update, nil)
		if err != nil {
			fmt.Printf("error storage.Adapter.UpdateAlertContact(%s) - %s", alertContact.ID, err)
			return fmt.Errorf("error storage.Adapter.UpdateAlertContact(%s) - %s", alertContact.ID, err)
		}
		if res.ModifiedCount != 1 {
			fmt.Printf("storage.Adapter.UpdateAlertContact(%s) invalid id", alertContact.ID)
			return fmt.Errorf("storage.Adapter.UpdateAlertContact(%s) invalid id", alertContact.ID)
		}
	}

	return nil
}

// DeleteAlertContact deletes an alert contact
func (sa *Adapter) DeleteAlertContact(user *model.User, id string) error {
	filter := bson.M{"_id": id, "org_id": user.Claims.OrgID, "app_id": user.Claims.AppID}
	res, err := sa.db.alertContacts.DeleteOne(filter, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.DeleteAlertContact(%s) - %s", id, err)
		return fmt.Errorf("error storage.Adapter.DeleteAlertContact(): error while delete alert contact (%s) - %s", id, err)
	}
	if res.DeletedCount != 1 {
		fmt.Printf("storage.Adapter.DeleteAlertContact(%s) invalid id", id)
		return fmt.Errorf("storage.Adapter.DeleteAlertContact(%s) invalid id", id)
	}
	return nil
}

// GetAllPolls gets all polls
func (sa *Adapter) GetAllPolls() ([]model.Poll, error) {
	filter := bson.M{}
	var results []model.Poll
	err := sa.db.polls.Find(filter, &results, nil)
	if err != nil {
		fmt.Printf("error storage.Adapter.GetAllPolls - %s", err)
		return nil, fmt.Errorf("error storage.Adapter.GetAllPolls - %s", err)
	}

	return results, nil
}
