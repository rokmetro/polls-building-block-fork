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

package core

import (
	"encoding/json"
	"fmt"
	"log"
	"polls/core/model"
	"polls/driven/groups"
	"polls/driven/storage"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (app *Application) getVersion() string {
	return app.version
}

func (app *Application) getPolls(user *model.User, filter model.PollsFilter, filterByToMembers bool) ([]model.Poll, error) {

	var membership *groups.GroupMembership
	if len(filter.GroupIDs) > 0 {
		groupMembership, err := app.groups.GetGroupsMembership(user.Token)
		if err != nil {
			log.Printf("error app.getPolls() - unable to retrieve user groups - %s", err)
			return nil, fmt.Errorf("error app.getPolls() - unable to retrieve user groups - %s", err)
		}

		membership = groupMembership
	}

	return app.storage.GetPolls(user, filter, filterByToMembers, membership)
}

func (app *Application) getPoll(user *model.User, id string) (*model.Poll, error) {
	groupMembership, err := app.groups.GetGroupsMembership(user.Token)
	if err != nil {
		log.Printf("error app.getPoll() - unable to retrieve user groups - %s", err)
		return nil, fmt.Errorf("error app.getPoll() - unable to retrieve user groups - %s", err)
	}

	return app.storage.GetPoll(user, id, true, groupMembership)
}

func (app *Application) createPoll(user *model.User, poll model.Poll) (*model.Poll, error) {
	createdPoll, err := app.storage.CreatePoll(user, poll)
	if err != nil {
		return nil, err
	}

	app.notifyNotificationsBBForPoll(user, createdPoll, "polls", "poll_created", fmt.Sprintf("Poll '%s' has been created", createdPoll.Question))

	if poll.GroupID != nil {
		go app.groups.UpdateGroupDateUpdated(*poll.GroupID)
	}

	return createdPoll, nil
}

func (app *Application) updatePoll(user *model.User, poll model.Poll) (*model.Poll, error) {
	persistedPoll, err := app.storage.GetPoll(user, poll.ID.Hex(), true, nil)
	if err != nil {
		return nil, err
	}

	err = app.checkPollPermission(user, persistedPoll, "update")
	if err != nil {
		return nil, err
	}

	updatedPoll, err := app.storage.UpdatePoll(user, poll)
	if err != nil {
		return nil, err
	}

	return updatedPoll, nil
}

func (app *Application) deletePoll(user *model.User, id string) error {
	poll, err := app.storage.GetPoll(user, id, true, nil)
	if err != nil {
		return err
	}

	err = app.checkPollPermission(user, poll, "delete")
	if err != nil {
		return err
	}

	err = app.storage.DeletePoll(user, id)
	if err != nil {
		return err
	}

	app.sseServer.NotifyPollForEvent(id, "poll_deleted")
	app.sseServer.ClosePoll(id)

	return nil
}

func (app *Application) startPoll(user *model.User, pollID string) error {
	poll, err := app.storage.GetPoll(user, pollID, true, nil)
	if err != nil {
		return err
	}

	err = app.checkPollPermission(user, poll, "start")
	if err != nil {
		return err
	}

	if poll != nil {
		if poll.Status != storage.PollStatusStarted {
			poll.Status = storage.PollStatusStarted
			_, err = app.storage.UpdatePoll(user, *poll)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("error app.startPoll() - poll not found: %s", pollID)
	}

	app.notifyNotificationsBBForPoll(user, poll, "polls", "poll_started", fmt.Sprintf("Poll '%s' has been started", poll.Question))

	app.sseServer.NotifyPollForEvent(pollID, "poll_started")

	if poll.GroupID != nil {
		go app.groups.UpdateGroupDateUpdated(*poll.GroupID)
	}

	return nil
}

func (app *Application) endPoll(user *model.User, pollID string) error {
	poll, err := app.storage.GetPoll(user, pollID, true, nil)
	if err != nil {
		return err
	}

	err = app.checkPollPermission(user, poll, "end")
	if err != nil {
		return err
	}

	if poll != nil {
		if poll.Status != storage.PollStatusTerminated {
			poll.Status = storage.PollStatusTerminated
			_, err = app.storage.UpdatePoll(user, *poll)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("error app.startPoll() - poll not found: %s", pollID)
	}

	app.notifyNotificationsBBForPoll(user, poll, "polls", "poll_ended", fmt.Sprintf("Poll '%s' has ended.", poll.Question))

	app.sseServer.NotifyPollForEvent(pollID, "poll_end")
	app.sseServer.ClosePoll(pollID)

	if poll.GroupID != nil {
		go app.groups.UpdateGroupDateUpdated(*poll.GroupID)
	}

	return nil
}

func (app *Application) notifyNotificationsBBForPoll(user *model.User, poll *model.Poll, topic string, operation string, message string) {
	subject := "Poll"
	if poll.GroupID != nil {

		group, _ := app.groups.GetGroupDetails(user.Token, *poll.GroupID)
		if group != nil {
			subject = fmt.Sprintf("Group Poll - %s", group.Title)
		}

		topic = fmt.Sprintf("group.%s", topic)
		app.groups.SendGroupNotification(*poll.GroupID, model.GroupNotification{
			Members: poll.ToMembersList.ToNotificationRecipients(),
			Sender: &model.Sender{
				Type: "user",
				User: &model.UserRef{
					UserID: user.Claims.Subject,
					Name:   user.Claims.Name,
				},
			},
			Topic:   &topic,
			Subject: subject,
			Body:    message,
			Data: map[string]string{
				"group_id":    *poll.GroupID,
				"type":        "poll",
				"operation":   operation,
				"entity_type": "poll",
				"entity_id":   poll.ID.Hex(),
				"entity_name": poll.Question,
			},
		})
	} else {
		app.notifications.SendNotification(model.NotificationMessage{
			Message: model.InnerMessage{
				AppID:      user.Claims.AppID,
				OrgID:      user.Claims.OrgID,
				Recipients: poll.ToMembersList.ToNotificationRecipients(),
				Sender: &model.Sender{
					Type: "user",
					User: &model.UserRef{
						UserID: user.Claims.Subject,
						Name:   user.Claims.Name,
					},
				},
				Topic:   &topic,
				Subject: subject,
				Body:    message,
				Data: map[string]string{
					"type":        "poll",
					"operation":   operation,
					"entity_type": "poll",
					"entity_id":   poll.ID.Hex(),
					"entity_name": poll.Question,
				},
			},
		})
	}
}

func (app *Application) votePoll(user *model.User, pollID string, vote model.PollVote) error {
	return app.storage.VotePoll(user, pollID, vote)
}

func (app *Application) subscribeToPoll(user *model.User, pollID string, resultChan chan map[string]interface{}) error {
	app.sseServer.RegisterUserForPoll(user.Claims.Subject, pollID, resultChan)
	return nil
}

func (app *Application) checkPollPermission(user *model.User, poll *model.Poll, operation string) error {
	if poll != nil {
		if user.Claims.Subject != poll.UserID {
			if poll.GroupID != nil && len(*poll.GroupID) > 0 {
				group, err := app.groups.GetGroupDetails(user.Token, *poll.GroupID)
				if err != nil {
					return err
				}
				if group != nil {
					if !group.IsCurrentUserAdmin(user.Claims.Subject) {
						return fmt.Errorf("only the creator of a poll or a group admin can %s it", operation)
					}
				} else {
					return fmt.Errorf("only the creator of a poll or a group admin can %s it", operation)
				}
			} else {
				return fmt.Errorf("only the creator of a poll can %s it", operation)
			}
		}
	} else {
		return fmt.Errorf("poll is nil")
	}
	return nil
}

// OnCollectionUpdated callback that indicates the reward types collection is changed
func (app *Application) OnCollectionUpdated(collection string, record map[string]interface{}) {
	if "polls" == collection && record != nil {
		data, err := json.Marshal(record)
		if err != nil {
			log.Printf("Error on Application.OnCollectionUpdated: %s", err)
			return
		}

		if data != nil {
			var poll model.PollNotification
			err = json.Unmarshal(data, &poll)
			if err != nil {
				log.Printf("Error on Application.OnCollectionUpdated: %s", err)
				return
			}

			app.sseServer.NotifyPollUpdate(poll.ID.Hex(), poll)
		}
	}
}

func (app *Application) getSurvey(user *model.User, id string) (*model.Survey, error) {
	return app.storage.GetSurvey(user, id)
}

func (app *Application) createSurvey(user *model.User, survey model.Survey, admin bool) (*model.Survey, error) {
	survey.ID = uuid.NewString()
	survey.CreatorID = user.Claims.Subject
	survey.DateCreated = time.Now().UTC()
	survey.AppID = user.Claims.AppID
	survey.OrgID = user.Claims.OrgID
	if !admin {
		survey.Type = "user"
	}
	return app.storage.CreateSurvey(survey)
}

func (app *Application) updateSurvey(user *model.User, survey model.Survey, id string, admin bool) error {
	survey.ID = id
	if !admin {
		survey.Type = "user"
	}
	return app.storage.UpdateSurvey(user, survey, admin)
}

func (app *Application) deleteSurvey(user *model.User, id string, admin bool) error {
	return app.storage.DeleteSurvey(user, id, admin)
}

func (app *Application) deleteSurveyResponses(user *model.User, surveyIDs []string, surveyTypes []string, startDate *time.Time, endDate *time.Time) error {
	return app.storage.DeleteSurveyResponses(user, surveyIDs, surveyTypes, startDate, endDate)
}

func (app *Application) getSurveyResponse(user *model.User, id string) (*model.SurveyResponse, error) {
	return app.storage.GetSurveyResponse(user, id)
}

func (app *Application) getSurveyResponses(user *model.User, surveyIDs []string, surveyTypes []string, startDate *time.Time, endDate *time.Time, limit *int, offset *int) ([]model.SurveyResponse, error) {
	return app.storage.GetSurveyResponses(user, surveyIDs, surveyTypes, startDate, endDate, limit, offset)
}

func (app *Application) createSurveyResponse(user *model.User, survey model.Survey) (*model.SurveyResponse, error) {
	response := model.SurveyResponse{ID: uuid.NewString(), AppID: user.Claims.AppID, OrgID: user.Claims.OrgID,
		UserID: user.Claims.Subject, DateCreated: time.Now().UTC(), Survey: survey}
	return app.storage.CreateSurveyResponse(response)
}

func (app *Application) updateSurveyResponse(user *model.User, id string, survey model.Survey) error {
	return app.storage.UpdateSurveyResponse(user, id, survey)
}

func (app *Application) deleteSurveyResponse(user *model.User, id string) error {
	return app.storage.DeleteSurveyResponse(user, id)
}

func (app *Application) getAlertContacts(user *model.User) ([]model.AlertContact, error) {
	return app.storage.GetAlertContacts(user)
}

func (app *Application) getAlertContact(user *model.User, id string) (*model.AlertContact, error) {
	return app.storage.GetAlertContact(user, id)
}

func (app *Application) createAlertContact(user *model.User, alertContact model.AlertContact) (*model.AlertContact, error) {
	alertContact.ID = uuid.NewString()
	alertContact.AppID = user.Claims.AppID
	alertContact.OrgID = user.Claims.OrgID
	alertContact.DateCreated = time.Now().UTC()
	return app.storage.CreateAlertContact(alertContact)
}

func (app *Application) updateAlertContact(user *model.User, id string, alertContact model.AlertContact) error {
	return app.storage.UpdateAlertContact(user, id, alertContact)
}

func (app *Application) deleteAlertContact(user *model.User, id string) error {
	return app.storage.DeleteAlertContact(user, id)
}

func (app *Application) createSurveyAlert(user *model.User, surveyAlert model.SurveyAlert) error {
	contacts, err := app.storage.GetAlertContactsByKey(surveyAlert.ContactKey, user)

	if err != nil {
		return err
	}

	for i := 0; i < len(contacts); i++ {
		if contacts[i].Type == "email" {
			subject, ok := surveyAlert.Content["subject"].(string)
			if !ok {
				return fmt.Errorf("error on Application.createSurveyAlert: No subject available")
			}
			body, ok := surveyAlert.Content["body"].(string)
			if !ok {
				return fmt.Errorf("error on Application.createSurveyAlert: No body available")
			}
			app.notifications.SendMail(user, contacts[i].Address, subject, body)
		}
	}

	return nil
}

func (app *Application) getUserData(user *model.User) (*model.UserDataResponse, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 3) // Channel to handle errors
	defer close(errChan)

	// Declare response variables
	var pollsReponse []model.Poll
	var survey []model.Survey
	var surveyResponse []model.SurveyResponse

	// Fetch polls asynchronously
	wg.Add(1)
	go func() {
		defer wg.Done()
		polls, err := app.storage.GetAllPolls() // Pass context if storage methods accept it
		if err != nil {
			errChan <- err
			return
		}

		if polls != nil {
			for _, p := range polls {
				if p.UserID == user.Claims.Subject {
					pollsReponse = append(pollsReponse, p)
				}
			}
		}
	}()

	// Fetch surveys asynchronously
	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := app.storage.GetSurveysByUserID(user)
		if err != nil {
			errChan <- err
			return
		}
		survey = s
	}()

	// Fetch survey responses asynchronously
	wg.Add(1)
	go func() {
		defer wg.Done()
		sr, err := app.storage.GetSurveyResponseByUserID(user)
		if err != nil {
			errChan <- err
			return
		}
		surveyResponse = sr
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	// Build the final response
	userResponse := model.UserDataResponse{
		Poll:           pollsReponse,
		Surveys:        survey,
		SurveyResponse: surveyResponse,
	}

	return &userResponse, nil
}
