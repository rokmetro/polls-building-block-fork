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
	"context"
	"log"
	"time"

	"github.com/rokwire/rokwire-building-block-sdk-go/utils/logging/logs"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionListener listens for collection updates
type CollectionListener interface {
	OnCollectionUpdated(name string, record map[string]interface{})
}

type database struct {
	listener CollectionListener

	mongoDBAuth  string
	mongoDBName  string
	mongoTimeout time.Duration

	db       *mongo.Database
	dbClient *mongo.Client
	logger   *logs.Logger

	polls           *collectionWrapper
	settings        *collectionWrapper
	surveys         *collectionWrapper
	surveyResponses *collectionWrapper
	alertContacts   *collectionWrapper
}

func (m *database) start() error {

	log.Println("database -> start")

	//connect to the database
	clientOptions := options.Client().ApplyURI(m.mongoDBAuth)
	connectContext, cancel := context.WithTimeout(context.Background(), m.mongoTimeout)
	client, err := mongo.Connect(connectContext, clientOptions)
	cancel()
	if err != nil {
		return err
	}

	//ping the database
	pingContext, cancel := context.WithTimeout(context.Background(), m.mongoTimeout)
	err = client.Ping(pingContext, nil)
	cancel()
	if err != nil {
		return err
	}

	//apply checks
	db := client.Database(m.mongoDBName)

	//asign the db, db client and the collections
	m.db = db
	m.dbClient = client

	settings := &collectionWrapper{database: m, coll: db.Collection("pollsettings")}
	err = m.applySettingsChecks(settings)
	if err != nil {
		return err
	}

	polls := &collectionWrapper{database: m, coll: db.Collection("polls")}
	err = m.applyPollsChecks(polls)
	if err != nil {
		return err
	}
	go polls.Watch(nil)

	surveys := &collectionWrapper{database: m, coll: db.Collection("surveys")}
	err = m.applySurveysChecks(surveys)
	if err != nil {
		return err
	}

	surveyResponses := &collectionWrapper{database: m, coll: db.Collection("surveyresponses")}
	err = m.applySurveyResponsesChecks(surveyResponses)
	if err != nil {
		return err
	}

	alertContacts := &collectionWrapper{database: m, coll: db.Collection("alert_contacts")}
	err = m.applyAlertContactsChecks(surveyResponses)
	if err != nil {
		return err
	}

	m.polls = polls
	m.settings = settings
	m.surveys = surveys
	m.surveyResponses = surveyResponses
	m.alertContacts = alertContacts

	return nil
}

func (m *database) applyPollsChecks(posts *collectionWrapper) error {
	log.Println("apply polls checks.....")

	indexes, _ := posts.ListIndexes()
	indexMapping := map[string]interface{}{}
	if indexes != nil {

		for _, index := range indexes {
			name := index["name"].(string)
			indexMapping[name] = index
		}
	}

	if indexMapping["org_id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "org_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["poll.group_id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "poll.group_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["poll.status_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "poll.status", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["poll.pin_1_poll.status_1__id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "poll.pin", Value: 1},
				primitive.E{Key: "poll.status", Value: 1},
				primitive.E{Key: "_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["poll.userid_1_poll.status_1__id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "poll.userid", Value: 1},
				primitive.E{Key: "poll.status", Value: 1},
				primitive.E{Key: "_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["responses.userid_1_poll.status_1__id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "responses.userid", Value: 1},
				primitive.E{Key: "poll.status", Value: 1},
				primitive.E{Key: "_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	log.Println("polls checks passed")
	return nil
}

func (m *database) applySettingsChecks(posts *collectionWrapper) error {
	log.Println("apply settings checks.....")

	indexes, _ := posts.ListIndexes()
	indexMapping := map[string]interface{}{}
	if indexes != nil {

		for _, index := range indexes {
			name := index["name"].(string)
			indexMapping[name] = index
		}
	}

	if indexMapping["org_id_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "org_id", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	if indexMapping["stadium_1"] == nil {
		err := posts.AddIndex(
			bson.D{
				primitive.E{Key: "stadium", Value: 1},
			}, false)
		if err != nil {
			return err
		}
	}

	log.Println("polls settings passed")
	return nil
}

func (m *database) applySurveysChecks(surveys *collectionWrapper) error {
	log.Println("apply surveys checks.....")

	err := surveys.AddIndex(bson.D{primitive.E{Key: "org_id", Value: 1}, primitive.E{Key: "app_id", Value: 1}, primitive.E{Key: "creator_id", Value: 1}}, false)
	if err != nil {
		return err
	}

	log.Println("surveys passed")
	return nil
}

func (m *database) applySurveyResponsesChecks(surveyResponses *collectionWrapper) error {
	log.Println("apply survey responses checks.....")

	err := surveyResponses.AddIndex(bson.D{primitive.E{Key: "org_id", Value: 1}, primitive.E{Key: "app_id", Value: 1}, primitive.E{Key: "user_id", Value: 1}}, false)
	if err != nil {
		return err
	}

	err = surveyResponses.AddIndex(bson.D{primitive.E{Key: "survey._id", Value: 1}}, false)
	if err != nil {
		return err
	}

	log.Println("survey responses passed")
	return nil
}

func (m *database) applyAlertContactsChecks(alertContacts *collectionWrapper) error {
	log.Println("apply alert contacts checks.....")

	err := alertContacts.AddIndex(bson.D{primitive.E{Key: "org_id", Value: 1}, primitive.E{Key: "app_id", Value: 1}, primitive.E{Key: "key", Value: 1}}, false)
	if err != nil {
		return err
	}

	log.Println("survey alert contacts passed")
	return nil
}
