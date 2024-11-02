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

package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"polls/core"
	"polls/core/model"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const maxUploadSize = 15 * 1024 * 1024 // 15 mb

// ApisHandler handles the rest APIs implementation
type ApisHandler struct {
	app    *core.Application
	config *model.Config
}

// Version gives the service version
// @Description Gives the service version.
// @Tags Client
// @ID Version
// @Produce plain
// @Success 200
// @Router /version [get]
func (h ApisHandler) Version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h.app.Services.GetVersion()))
}

// NewApisHandler creates new rest Handler instance
func NewApisHandler(app *core.Application, config *model.Config) ApisHandler {
	return ApisHandler{app: app, config: config}
}

// NewAdminApisHandler creates new rest Handler instance
func NewAdminApisHandler(app *core.Application, config *model.Config) AdminApisHandler {
	return AdminApisHandler{app: app, config: config}
}

// NewInternalApisHandler creates new rest Handler instance
func NewInternalApisHandler(app *core.Application, config *model.Config) InternalApisHandler {
	return InternalApisHandler{app: app, config: config}
}

// GetPolls Retrieves  all polls by a filter params
// @Description Retrieves  all polls by a filter params
// @Tags Client
// @ID GetPolls
// @Success 200 {array} model.PollResult
// @Security UserAuth
// @Router /polls [get]
func (h ApisHandler) GetPolls(user *model.User, w http.ResponseWriter, r *http.Request) {

	var filter model.PollsFilter

	pinRaw := r.URL.Query().Get("pin")
	if len(pinRaw) > 0 {
		intParsed, err := strconv.Atoi(pinRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetPolls: invalid pin - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filter.Pin = &intParsed
	}
	myPollsRaw := r.URL.Query().Get("my_polls")
	if len(myPollsRaw) > 0 {
		boolParsed, err := strconv.ParseBool(myPollsRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetPolls: invalid my_polls - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filter.MyPolls = &boolParsed
	}
	respondedPollsRaw := r.URL.Query().Get("responded_polls")
	if len(respondedPollsRaw) > 0 {
		boolParsed, err := strconv.ParseBool(respondedPollsRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetPolls: invalid responded_polls - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filter.RespondedPolls = &boolParsed
	}

	pollIDsRaw := r.URL.Query().Get("poll_ids")
	if len(pollIDsRaw) > 0 {
		filter.PollIDs = strings.Split(pollIDsRaw, ",")
	}
	groupIDsRaw := r.URL.Query().Get("group_ids")
	if len(groupIDsRaw) > 0 {
		filter.GroupIDs = strings.Split(groupIDsRaw, ",")
	}
	statusesRaw := r.URL.Query().Get("statuses")
	if len(statusesRaw) > 0 {
		filter.Statuses = strings.Split(statusesRaw, ",")
	}

	limitRaw := r.URL.Query().Get("limit")
	limit := int64(20)
	if len(limitRaw) > 0 {
		intParsed, err := strconv.Atoi(limitRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetPolls: invalid limit - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		limit = int64(intParsed)
	}
	filter.Limit = &limit

	offsetRaw := r.URL.Query().Get("offset")
	offset := int64(0)
	if len(offsetRaw) > 0 {
		intParsed, err := strconv.Atoi(offsetRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetPolls: invalid offset - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		offset = int64(intParsed)
	}
	filter.Offset = &offset

	resData, err := h.app.Services.GetPolls(user, filter, true)
	if err != nil {
		log.Printf("Error on apis.GetPolls(): %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	result := []model.PollResult{}
	if len(resData) > 0 {
		for _, entry := range resData {
			result = append(result, entry.ToPollResult(user.Claims.Subject))
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error on apis.GetPolls(): %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// GetPoll Retrieves a poll by id
// @Description Retrieves a poll by id
// @Tags Client
// @ID GetPoll
// @Accept json
// @Produce json
// @Success 200 {object} model.Poll
// @Failure 401
// @Security UserAuth
// @Router /polls/{id} [get]
func (h ApisHandler) GetPoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.GetPoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if resData == nil {
		log.Printf("Error on apis.VotePoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data, err := json.Marshal(resData.ToPollResult(user.Claims.Subject))
	if err != nil {
		log.Printf("Error on apis.GetPoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// UpdatePoll Updates a reward type with the specified id
// @Description Updates a reward type with the specified id
// @Tags Client
// @ID UpdatePoll
// @Param data body model.Poll true "body json"
// @Accept json
// @Produce json
// @Success 200 {object} model.Poll
// @Failure 401
// @Security UserAuth
// @Router /polls/{id} [put]
func (h ApisHandler) UpdatePoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.UpdatePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if resData == nil {
		log.Printf("Error on apis.VotePoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.UpdatePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Poll
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.UpdatePoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resData, err = h.app.Services.UpdatePoll(user, item)
	if err != nil {
		log.Printf("Error on apis.UpdatePoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(resData.ToPollResult(user.Claims.Subject))
	if err != nil {
		log.Printf("Error on apis.UpdatePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// CreatePoll Create a new poll
// @Description Create a new poll
// @Tags Client
// @ID CreatePoll
// @Param data body model.Poll true "body json"
// @Accept json
// @Success 200 {object} model.Poll
// @Security UserAuth
// @Router /polls [post]
func (h ApisHandler) CreatePoll(user *model.User, w http.ResponseWriter, r *http.Request) {

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.CreatePoll: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Poll
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.CreatePoll: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdItem, err := h.app.Services.CreatePoll(user, item)
	if err != nil {
		log.Printf("Error on apis.CreatePoll: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(createdItem.ToPollResult(user.Claims.Subject))
	if err != nil {
		log.Printf("Error on apis.CreatePoll: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// DeletePoll Deletes a poll with the specified id
// @Description Deletes a poll with the specified id
// @Tags Client
// @ID DeletePoll
// @Success 200
// @Security UserAuth
// @Router /polls/{id} [delete]
func (h ApisHandler) DeletePoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.DeletePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if resData == nil {
		log.Printf("Error on apis.DeletePoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	err = h.app.Services.DeletePoll(user, id)
	if err != nil {
		log.Printf("Error on apis.DeletePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// GetPollEvents Subscribes to a poll events as SSE
// @Description  Subscribes to a poll events as SSE
// @Tags Client
// @ID GetPollEvents
// @Produce json
// @Success 200
// @Security UserAuth
// @Router /polls/{id}/events [post]
func (h ApisHandler) GetPollEvents(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.GetPollEvents(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if resData == nil {
		log.Printf("Error on apis.GetPollEvents(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Connection doesn't support streaming", http.StatusBadRequest)
		return
	}

	resultChan := make(chan map[string]interface{})

	go h.app.Services.SubscribeToPoll(user, id, resultChan)

	for {
		data, ok := <-resultChan
		if ok {
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error on apis.GetPollEvents(): %s", err)
			}
			log.Printf(string(jsonData))
			w.Write(jsonData)
			flusher.Flush()
		} else {
			flusher.Flush()
			break
		}
	}
	log.Printf("closing event stream for user %s and poll %s", user.Claims.Subject, id)
}

// VotePoll Votes a poll with the specified id
// @Description  Votes a poll with the specified id
// @Tags Client
// @ID VotePoll
// @Param data body model.PollVote true "body json"
// @Accept json
// @Produce json
// @Success 200
// @Security UserAuth
// @Router /polls/{id}/vote [post]
func (h ApisHandler) VotePoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.VotePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if resData == nil {
		log.Printf("Error on apis.VotePoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.VotePoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.PollVote
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.VotePoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if user.Claims.Subject != item.UserID {
		log.Printf("Error on apis.VotePoll(%s): inconsistent user id", id)
		http.Error(w, "inconsistent user id", http.StatusBadRequest)
	}

	err = h.app.Services.VotePoll(user, id, item)
	if err != nil {
		log.Printf("Error on apis.VotePoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// StartPoll Starts an existing poll with the specified id
// @Description  Starts an existing poll with the specified id
// @Tags Client
// @ID StartPoll
// @Accept json
// @Produce json
// @Success 200
// @Security UserAuth
// @Router /polls/{id}/start [post]
func (h ApisHandler) StartPoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.StartPoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if resData == nil {
		log.Printf("Error on apis.StartPoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	err = h.app.Services.StartPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.StartPoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// EndPoll Finishes an existing poll with the specified id
// @Description  Finishes an existing poll with the specified id
// @Tags Client
// @ID EndPoll
// @Accept json
// @Produce json
// @Success 200
// @Security UserAuth
// @Router /polls/{id}/end [post]
func (h ApisHandler) EndPoll(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.EndPoll(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if resData == nil {
		log.Printf("Error on apis.EndPoll(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	err = h.app.Services.EndPoll(user, id)
	if err != nil {
		log.Printf("Error on apis.EndPoll(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// GetSurvey Retrieves a Survey by id
// @Description Retrieves a Survey by id
// @Tags Client
// @ID GetSurvey
// @Accept json
// @Produce json
// @Success 200 {object} model.Survey
// @Failure 401
// @Security UserAuth
// @Router /surveys/{id} [get]
func (h ApisHandler) GetSurvey(user *model.User, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetSurvey(user, id)
	if err != nil {
		log.Printf("Error on apis.GetSurvey(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resData == nil {
		log.Printf("Error on apis.GetSurvey(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data, err := json.Marshal(resData)
	if err != nil {

		log.Printf("Error on apis.GetSurvey(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// CreateSurvey Create a new survey
// @Description Create a new survey
// @Tags Client
// @ID CreateSurvey
// @Param data body model.Survey true "body json"
// @Accept json
// @Success 200 {object} model.Survey
// @Security UserAuth
// @Router /surveys [post]
func (h ApisHandler) CreateSurvey(user *model.User, w http.ResponseWriter, r *http.Request) {

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.CreateSurvey: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Survey
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.CreateSurvey: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdItem, err := h.app.Services.CreateSurvey(user, item, false)
	if err != nil {
		log.Printf("Error on apis.CreateSurvey: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(createdItem)
	if err != nil {
		log.Printf("Error on apis.CreateSurvey: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// UpdateSurvey Updates a survey type with the specified id
// @Description Updates a survey type with the specified id
// @Tags Client
// @ID UpdateSurvey
// @Param data body model.Survey true "body json"
// @Accept json
// @Produce json
// @Success 200 {object} model.Survey
// @Failure 401
// @Security UserAuth
// @Router /surveys/{id} [put]
func (h ApisHandler) UpdateSurvey(user *model.User, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {

		log.Printf("Error on apis.UpdateSurvey(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Survey
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.UpdateSurvey(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.app.Services.UpdateSurvey(user, item, id, false)
	if err != nil {
		log.Printf("Error on apis.UpdateSurvey(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// DeleteSurvey Deletes a survey with the specified id
// @Description Deletes a survey with the specified id
// @Tags Client
// @ID DeleteSurvey
// @Success 200
// @Security UserAuth
// @Router /surveys/{id} [delete]
func (h ApisHandler) DeleteSurvey(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.app.Services.DeleteSurvey(user, id, false)
	if err != nil {
		log.Printf("Error on apis.DeleteSurvey(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// GetSurveyResponses retrieves SurveyResponses for the current user
// @Description Retrieves SurveyResponses for the current user
// @Tags Client
// @ID GetSurveyResponse
// @Accept json
// @Produce json
// @Success 200 {array} model.SurveyResponse
// @Failure 401
// @Security UserAuth
// @Router /survey-responses [get]
func (h ApisHandler) GetSurveyResponses(user *model.User, w http.ResponseWriter, r *http.Request) {
	surveyIDsRaw := r.URL.Query().Get("survey_ids")
	var surveyIDs []string
	if len(surveyIDsRaw) > 0 {
		surveyIDs = strings.Split(surveyIDsRaw, ",")
	}
	surveyTypesRaw := r.URL.Query().Get("survey_types")
	var surveyTypes []string
	if len(surveyTypesRaw) > 0 {
		surveyTypes = strings.Split(surveyTypesRaw, ",")
	}
	startDateRaw := r.URL.Query().Get("start_date")
	var startDate *time.Time
	if len(startDateRaw) > 0 {
		dateParsed, err := time.Parse(time.RFC3339, startDateRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetSurveyResponses: invalid start date - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		startDate = &dateParsed
	}
	endDateRaw := r.URL.Query().Get("end_date")
	var endDate *time.Time
	if len(endDateRaw) > 0 {
		dateParsed, err := time.Parse(time.RFC3339, endDateRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetSurveyResponses: invalid end date - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		endDate = &dateParsed
	}

	limitRaw := r.URL.Query().Get("limit")
	limit := 20
	if len(limitRaw) > 0 {
		intParsed, err := strconv.Atoi(limitRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetSurveyResponses: invalid limit - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		limit = intParsed
	}
	offsetRaw := r.URL.Query().Get("offset")
	offset := 0
	if len(offsetRaw) > 0 {
		intParsed, err := strconv.Atoi(offsetRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.GetSurveyResponses: invalid offset - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		offset = intParsed
	}

	resData, err := h.app.Services.GetSurveyResponses(user, surveyIDs, surveyTypes, startDate, endDate, &limit, &offset)
	if err != nil {
		log.Printf("Error on apis.GetSurveyResponses: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(resData)
	if err != nil {
		log.Printf("Error on apis.GetSurveyResponse: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// GetSurveyResponse Retrieves a SurveyResponse by id
// @Description Retrieves a SurveyResponse by id
// @Tags Client
// @ID GetSurveyResponse
// @Accept json
// @Produce json
// @Success 200 {object} model.SurveyResponse
// @Failure 401
// @Security UserAuth
// @Router /survey-responses/{id} [get]
func (h ApisHandler) GetSurveyResponse(user *model.User, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	resData, err := h.app.Services.GetSurveyResponse(user, id)
	if err != nil {
		log.Printf("Error on apis.GetSurveyResponse(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resData == nil {
		log.Printf("Error on apis.GetSurveyResponse(%s): not found", id)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data, err := json.Marshal(resData)
	if err != nil {
		log.Printf("Error on apis.GetSurveyResponse(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// CreateSurveyResponse Create a new survey response
// @Description Create a new survey response
// @Tags Client
// @ID CreateSurveyResponse
// @Param data body model.Survey true "body json"
// @Accept json
// @Success 200 {object} model.SurveyResponse
// @Security UserAuth
// @Router /survey-responses [post]
func (h ApisHandler) CreateSurveyResponse(user *model.User, w http.ResponseWriter, r *http.Request) {

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyResponse: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Survey
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyResponse: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdItem, err := h.app.Services.CreateSurveyResponse(user, item)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyResponse: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(createdItem)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyResponse: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// UpdateSurveyResponse Updates a survey response type with the specified id
// @Description Updates a survey response type with the specified id
// @Tags Client
// @ID UpdateSurveyResponse
// @Param data body model.Survey true "body json"
// @Accept json
// @Produce json
// @Success 200 {object} model.SurveyResponse
// @Failure 401
// @Security UserAuth
// @Router /survey-responses/{id} [put]
func (h ApisHandler) UpdateSurveyResponse(user *model.User, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {

		log.Printf("Error on apis.UpdateSurveyResponse(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.Survey
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.UpdateSurveyResponse(%s): %s", id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.app.Services.UpdateSurveyResponse(user, id, item)
	if err != nil {
		log.Printf("Error on apis.DeleteSurveyResponse(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// DeleteSurveyResponse Deletes a survey response with the specified id
// @Description Deletes a survey response with the specified id
// @Tags Client
// @ID DeleteSurveyResponse
// @Success 200
// @Security UserAuth
// @Router /survey-responses/{id} [delete]
func (h ApisHandler) DeleteSurveyResponse(user *model.User, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.app.Services.DeleteSurveyResponse(user, id)
	if err != nil {
		log.Printf("Error on apis.DeleteSurveyResponse(%s): %s", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// DeleteSurveyResponses delete SurveyResponses for the current user
// @Description Deletes SurveyResponses for the current user
// @Tags Client
// @ID DeleteSurveyResponses
// @Accept json
// @Success 200
// @Failure 401
// @Security UserAuth
// @Router /survey-responses [delete]
func (h ApisHandler) DeleteSurveyResponses(user *model.User, w http.ResponseWriter, r *http.Request) {

	surveyIDsRaw := r.URL.Query().Get("survey_ids")
	var surveyIDs []string
	if len(surveyIDsRaw) > 0 {
		surveyIDs = strings.Split(surveyIDsRaw, ",")
	}
	surveyTypesRaw := r.URL.Query().Get("survey_types")
	var surveyTypes []string
	if len(surveyTypesRaw) > 0 {
		surveyTypes = strings.Split(surveyTypesRaw, ",")
	}
	startDateRaw := r.URL.Query().Get("start_date")
	var startDate *time.Time
	if len(startDateRaw) > 0 {
		dateParsed, err := time.Parse(time.RFC3339, startDateRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.DeleteSurveyResponses: invalid start date - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		startDate = &dateParsed
	}
	endDateRaw := r.URL.Query().Get("end_date")
	var endDate *time.Time
	if len(endDateRaw) > 0 {
		dateParsed, err := time.Parse(time.RFC3339, endDateRaw)
		if err != nil {
			err = fmt.Errorf("error on apis.DeleteSurveyResponses: invalid end date - %v", err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		endDate = &dateParsed
	}

	err := h.app.Services.DeleteSurveyResponses(user, surveyIDs, surveyTypes, startDate, endDate)
	if err != nil {
		log.Printf("Error on apis.DeleteSurveyResponses: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// CreateSurveyAlert Creates a survey alert
// @Description Create a new survey alert to be sent to notifications BB
// @Tags Client
// @ID CreateSurveyAlert
// @Param data body model.SurveyAlert true "body json"
// @Accept json
// @Success 200 {object} model.SurveyAlert
// @Security UserAuth
// @Router /survey-alert [post]
func (h ApisHandler) CreateSurveyAlert(user *model.User, w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on apis.CreateAlert: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var item model.SurveyAlert
	err = json.Unmarshal(data, &item)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyAlert: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.app.Services.CreateSurveyAlert(user, item)
	if err != nil {
		log.Printf("Error on apis.CreateSurveyAlert: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
