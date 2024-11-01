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
	"net/http"
	"polls/core"
	"polls/core/model"
	"polls/driver/web/rest"

	"github.com/rokwire/core-auth-library-go/v3/authservice"
	"github.com/rokwire/core-auth-library-go/v3/webauth"
	"github.com/rokwire/logging-library-go/v2/logs"

	"github.com/casbin/casbin"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Adapter entity
type Adapter struct {
	host          string
	port          string
	auth          *Auth
	authorization *casbin.Enforcer

	apisHandler         rest.ApisHandler
	adminApisHandler    rest.AdminApisHandler
	internalApisHandler rest.InternalApisHandler

	corsAllowedOrigins []string
	corsAllowedHeaders []string

	app    *core.Application
	logger *logs.Logger
}

// @title Polls Building Block v2 API
// @description RoRewards Building Block API Documentation.
// @version 1.0.21
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost
// @BasePath /content
// @schemes https

// @securityDefinitions.apikey InternalApiAuth
// @in header (add INTERNAL-API-KEY with correct value as a header)
// @name Authorization

// @securityDefinitions.apikey AdminUserAuth
// @in header (add Bearer prefix to the Authorization value)
// @name Authorization

// @securityDefinitions.apikey AdminGroupAuth
// @in header
// @name GROUP

// Start starts the module
func (we Adapter) Start() {

	router := mux.NewRouter().StrictSlash(true)
	router.Use()

	subrouter := router.PathPrefix("/polls").Subrouter()
	subrouter.PathPrefix("/doc/ui").Handler(we.serveDocUI())
	subrouter.HandleFunc("/doc", we.serveDoc)
	subrouter.HandleFunc("/version", we.wrapFunc(we.apisHandler.Version)).Methods("GET")

	// handle apis
	apiRouter := subrouter.PathPrefix("/api").Subrouter()

	// Client APIs
	apiRouter.HandleFunc("/polls", we.userAuthWrapFunc(we.apisHandler.GetPolls)).Methods("GET")
	apiRouter.HandleFunc("/polls", we.userAuthWrapFunc(we.apisHandler.CreatePoll)).Methods("POST")
	apiRouter.HandleFunc("/polls/{id}", we.userAuthWrapFunc(we.apisHandler.GetPoll)).Methods("GET")
	apiRouter.HandleFunc("/polls/{id}", we.userAuthWrapFunc(we.apisHandler.UpdatePoll)).Methods("PUT")
	apiRouter.HandleFunc("/polls/{id}", we.userAuthWrapFunc(we.apisHandler.DeletePoll)).Methods("DELETE")
	apiRouter.HandleFunc("/polls/{id}/events", we.userAuthWrapFunc(we.apisHandler.GetPollEvents)).Methods("GET")
	apiRouter.HandleFunc("/polls/{id}/vote", we.userAuthWrapFunc(we.apisHandler.VotePoll)).Methods("PUT")
	apiRouter.HandleFunc("/polls/{id}/start", we.userAuthWrapFunc(we.apisHandler.StartPoll)).Methods("PUT")
	apiRouter.HandleFunc("/polls/{id}/end", we.userAuthWrapFunc(we.apisHandler.EndPoll)).Methods("PUT")
	apiRouter.HandleFunc("/surveys/{id}", we.userAuthWrapFunc(we.apisHandler.GetSurvey)).Methods("GET")
	apiRouter.HandleFunc("/surveys", we.userAuthWrapFunc(we.apisHandler.CreateSurvey)).Methods("POST")
	apiRouter.HandleFunc("/surveys/{id}", we.userAuthWrapFunc(we.apisHandler.UpdateSurvey)).Methods("PUT")
	apiRouter.HandleFunc("/surveys/{id}", we.userAuthWrapFunc(we.apisHandler.DeleteSurvey)).Methods("DELETE")
	apiRouter.HandleFunc("/survey-responses/{id}", we.userAuthWrapFunc(we.apisHandler.GetSurveyResponse)).Methods("GET")
	apiRouter.HandleFunc("/survey-responses", we.userAuthWrapFunc(we.apisHandler.GetSurveyResponses)).Methods("GET")
	apiRouter.HandleFunc("/survey-responses", we.userAuthWrapFunc(we.apisHandler.CreateSurveyResponse)).Methods("POST")
	apiRouter.HandleFunc("/survey-responses/{id}", we.userAuthWrapFunc(we.apisHandler.UpdateSurveyResponse)).Methods("PUT")
	apiRouter.HandleFunc("/survey-responses/{id}", we.userAuthWrapFunc(we.apisHandler.DeleteSurveyResponse)).Methods("DELETE")
	apiRouter.HandleFunc("/survey-responses", we.userAuthWrapFunc(we.apisHandler.DeleteSurveyResponses)).Methods("DELETE")
	apiRouter.HandleFunc("/survey-alerts", we.userAuthWrapFunc(we.apisHandler.CreateSurveyAlert)).Methods("POST")

	// handle admin apis
	adminRouter := apiRouter.PathPrefix("/admin").Subrouter()

	adminRouter.HandleFunc("/surveys/{id}", we.adminAuthWrapFunc(we.adminApisHandler.GetSurvey)).Methods("GET")
	adminRouter.HandleFunc("/surveys", we.adminAuthWrapFunc(we.adminApisHandler.CreateSurvey)).Methods("POST")
	adminRouter.HandleFunc("/surveys/{id}", we.adminAuthWrapFunc(we.adminApisHandler.UpdateSurvey)).Methods("PUT")
	adminRouter.HandleFunc("/surveys/{id}", we.adminAuthWrapFunc(we.adminApisHandler.DeleteSurvey)).Methods("DELETE")
	adminRouter.HandleFunc("/alert-contacts", we.adminAuthWrapFunc(we.adminApisHandler.GetAlertContacts)).Methods("GET")
	adminRouter.HandleFunc("/alert-contacts/{id}", we.adminAuthWrapFunc(we.adminApisHandler.GetAlertContact)).Methods("GET")
	adminRouter.HandleFunc("/alert-contacts", we.adminAuthWrapFunc(we.adminApisHandler.CreateAlertContact)).Methods("POST")
	adminRouter.HandleFunc("/alert-contacts/{id}", we.adminAuthWrapFunc(we.adminApisHandler.UpdateAlertContact)).Methods("PUT")
	adminRouter.HandleFunc("/alert-contacts/{id}", we.adminAuthWrapFunc(we.adminApisHandler.DeleteAlertContact)).Methods("DELETE")

	var handler http.Handler = router
	if len(we.corsAllowedOrigins) > 0 {
		handler = webauth.SetupCORS(we.corsAllowedOrigins, we.corsAllowedHeaders, router)
	}
	we.logger.Fatalf("Error serving: %v", http.ListenAndServe(":"+we.port, handler))
}

func (we Adapter) serveDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("access-control-allow-origin", "*")
	http.ServeFile(w, r, "./driver/web/docs/gen/def.yaml")
}

func (we Adapter) serveDocUI() http.Handler {
	url := fmt.Sprintf("%s/polls/doc", we.host)
	return httpSwagger.Handler(httpSwagger.URL(url))
}

func (we Adapter) wrapFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logObj := we.logger.NewRequestLog(req)
		logObj.RequestReceived()
		defer logObj.RequestComplete()

		handler(w, req)
	}
}

type apiKeysAuthFunc = func(http.ResponseWriter, *http.Request)

func (we Adapter) apiKeyOrTokenWrapFunc(handler apiKeysAuthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logObj := we.logger.NewRequestLog(req)
		logObj.RequestReceived()
		defer logObj.RequestComplete()

		// apply core token check
		coreAuth, _ := we.auth.coreAuth.Check(req)
		if coreAuth {
			handler(w, req)
			return
		}

		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

type authFunc = func(*model.User, http.ResponseWriter, *http.Request)

func (we Adapter) userAuthWrapFunc(handler authFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logObj := we.logger.NewRequestLog(req)
		logObj.RequestReceived()
		defer logObj.RequestComplete()

		coreAuth, user := we.auth.coreAuth.Check(req)
		if coreAuth && user != nil && !user.Claims.Anonymous {
			handler(user, w, req)
			return
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

// TODO: Switch to Core BB model for auth
func (we Adapter) adminAuthWrapFunc(handler authFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logObj := we.logger.NewRequestLog(req)
		logObj.RequestReceived()
		defer logObj.RequestComplete()

		valid, hasAccess, user := we.auth.coreAuth.CheckWithAuthorization(req)
		if valid && hasAccess {
			handler(user, w, req)
			return
		}

		if !valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !hasAccess {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	}
}

type internalAPIKeyAuthFunc = func(http.ResponseWriter, *http.Request)

func (we Adapter) internalAPIKeyAuthWrapFunc(handler internalAPIKeyAuthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logObj := we.logger.NewRequestLog(req)
		logObj.RequestReceived()
		defer logObj.RequestComplete()

		apiKeyAuthenticated := we.auth.internalAuth.check(w, req)

		if apiKeyAuthenticated {
			handler(w, req)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

// NewWebAdapter creates new WebAdapter instance
func NewWebAdapter(host string, port string, app *core.Application, config *model.Config, serviceRegManager *authservice.ServiceRegManager,
	corsAllowedOrigins []string, corsAllowedHeaders []string, logger *logs.Logger) Adapter {
	auth := NewAuth(app, config, serviceRegManager, logger)
	authorization := casbin.NewEnforcer("driver/web/authorization_model.conf", "driver/web/authorization_policy.csv")

	apisHandler := rest.NewApisHandler(app, config)
	adminApisHandler := rest.NewAdminApisHandler(app, config)
	internalApisHandler := rest.NewInternalApisHandler(app, config)
	return Adapter{
		host:                host,
		port:                port,
		auth:                auth,
		authorization:       authorization,
		apisHandler:         apisHandler,
		adminApisHandler:    adminApisHandler,
		internalApisHandler: internalApisHandler,
		app:                 app,
		corsAllowedOrigins:  corsAllowedOrigins,
		corsAllowedHeaders:  corsAllowedHeaders,
		logger:              logger,
	}
}

// AppListener implements core.ApplicationListener interface
type AppListener struct {
	adapter *Adapter
}
