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

package web

import (
	"fmt"
	"log"
	"net/http"
	"polls/core"
	"polls/core/model"
	web "polls/driver/web/auth"

	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth"
	"github.com/rokwire/rokwire-building-block-sdk-go/utils/logging/logs"
)

// Auth handler
type Auth struct {
	internalAuth *InternalAuth
	coreAuth     *web.CoreAuth
	logger       *logs.Logger
}

func (auth *Auth) clientIDCheck(w http.ResponseWriter, r *http.Request) bool {
	clientID := r.Header.Get("APP")
	if len(clientID) == 0 {
		clientID = "edu.illinois.rokwire"
	}

	log.Println(fmt.Sprintf("400 - Bad Request"))
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
	return false
}

// NewAuth creates new auth handler
func NewAuth(app *core.Application, config *model.Config, serviceRegManager *auth.ServiceRegManager, logger *logs.Logger) *Auth {
	coreAuth := web.NewCoreAuth(app, serviceRegManager)
	internalAuth := newInternalAuth(config)
	auth := Auth{coreAuth: coreAuth, internalAuth: internalAuth, logger: logger}
	return &auth
}

// InternalAuth handling the internal calls fromother BBs
type InternalAuth struct {
	internalAPIKey string
}

func newInternalAuth(config *model.Config) *InternalAuth {
	auth := InternalAuth{internalAPIKey: config.InternalAPIKey}
	return &auth
}

func (auth *InternalAuth) check(w http.ResponseWriter, r *http.Request) bool {
	apiKey := r.Header.Get("INTERNAL-API-KEY")
	//check if there is api key in the header
	if len(apiKey) == 0 {
		//no key, so return 400
		log.Println(fmt.Sprintf("400 - Bad Request"))

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
		return false
	}

	exist := auth.internalAPIKey == apiKey

	if !exist {
		//not exist, so return 401
		log.Println(fmt.Sprintf("401 - Unauthorized for key %s", apiKey))

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
		return false
	}
	return true
}
