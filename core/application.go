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
	cacheadapter "polls/driven/cache"
	corebb "polls/driven/core"

	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth/tokenauth"

	"github.com/rokwire/rokwire-building-block-sdk-go/utils/logging/logs"

	"polls/driven/groups"
	"polls/driven/notifications"
)

// Application represents the core application code based on hexagonal architecture
type Application struct {
	version string
	build   string

	Services Services //expose to the drivers adapters
	logger   *logs.Logger

	storage       Storage
	cache         *cacheadapter.CacheAdapter
	notifications *notifications.Adapter
	groups        *groups.Adapter
	sseServer     *SSEServer
	tokenAuth     *tokenauth.TokenAuth

	serviceID       string
	corebb          *corebb.Adapter
	deleteDataLogic deleteDataLogic
}

// Start starts the core part of the application
func (app *Application) Start() {
	app.storage.SetListener(app)
	app.deleteDataLogic.start()
}

// NewApplication creates new Application
func NewApplication(version string, build string, storage Storage, cacheAdapter *cacheadapter.CacheAdapter,
	notificationsAdapter *notifications.Adapter, groupsAdapter *groups.Adapter, serviceID string, coreBB *corebb.Adapter, logger *logs.Logger) *Application {
	deleteDataLogic := deleteDataLogic{logger: *logger, core: coreBB, serviceID: serviceID, storage: storage}

	application := Application{
		version:         version,
		build:           build,
		storage:         storage,
		cache:           cacheAdapter,
		notifications:   notificationsAdapter,
		groups:          groupsAdapter,
		sseServer:       NewSSEServer(),
		serviceID:       serviceID,
		corebb:          coreBB,
		deleteDataLogic: deleteDataLogic,
	}

	// add the drivers ports/interfaces
	application.Services = &servicesImpl{app: &application}

	return &application
}
