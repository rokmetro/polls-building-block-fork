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

package main

import (
	"log"

	"polls/core"
	"polls/core/model"
	cacheadapter "polls/driven/cache"
	corebb "polls/driven/core"
	"polls/driven/groups"
	"polls/driven/notifications"
	storage "polls/driven/storage"
	driver "polls/driver/web"
	"strings"

	"github.com/rokwire/core-auth-library-go/v3/authservice"
	"github.com/rokwire/core-auth-library-go/v3/envloader"
	"github.com/rokwire/core-auth-library-go/v3/keys"
	"github.com/rokwire/core-auth-library-go/v3/sigauth"
	"github.com/rokwire/logging-library-go/v2/logs"
)

var (
	// Version : version of this executable
	Version string
	// Build : build date of this executable
	Build string
)

func main() {
	if len(Version) == 0 {
		Version = "dev"
	}

	serviceID := "polls-v2"
	envPrefix := "POLLS_"

	loggerOpts := logs.LoggerOpts{
		SensitiveHeaders: []string{"Rokwire-Api-Key", "User-ID", "Rokwire-Hs-Api-Key", "Group", "Rokwire-Acc-ID", "Csrf"},
		SuppressRequests: logs.NewStandardHealthCheckHTTPRequestProperties("polls/version"),
	}
	logger := logs.NewLogger(serviceID, &loggerOpts)
	envLoader := envloader.NewEnvLoader(Version, logger)

	port := envLoader.GetAndLogEnvVar("PORT", true, false)

	internalAPIKey := envLoader.GetAndLogEnvVar("INTERNAL_API_KEY", true, true)

	//mongoDB adapter
	mongoDBAuth := envLoader.GetAndLogEnvVar("MONGO_AUTH", true, true)
	mongoDBName := envLoader.GetAndLogEnvVar("MONGO_DATABASE", true, false)
	mongoTimeout := envLoader.GetAndLogEnvVar("MONGO_TIMEOUT", false, false)

	// web adapter
	host := envLoader.GetAndLogEnvVar("HOST", true, false)
	coreBBHost := envLoader.GetAndLogEnvVar("CORE_BB_HOST", true, false)
	serviceURL := envLoader.GetAndLogEnvVar("POLL_SERVICE_URL", true, false)
	uiucOrgID := envLoader.GetAndLogEnvVar("UIUC_ORG_ID", true, true)

	// Groups BB Host
	groupsBBHost := envLoader.GetAndLogEnvVar(envPrefix+"GROUPS_BB_HOST", true, false)

	// Notifications BB Host
	notificationsBBHost := envLoader.GetAndLogEnvVar(envPrefix+"NOTIFICATIONS_BB_HOST", true, false)

	authService := authservice.AuthService{
		ServiceID:   serviceID,
		ServiceHost: serviceURL,
		FirstParty:  true,
		AuthBaseURL: coreBBHost,
	}

	serviceRegLoader, err := authservice.NewRemoteServiceRegLoader(&authService, []string{"auth"})
	if err != nil {
		log.Fatalf("Error initializing remote service registration loader: %v", err)
	}

	serviceRegManager, err := authservice.NewServiceRegManager(&authService, serviceRegLoader, !strings.HasPrefix(serviceURL, "http://localhost"))
	if err != nil {
		log.Fatalf("Error initializing service registration manager: %v", err)
	}

	//core adapter
	var serviceAccountManager *authservice.ServiceAccountManager

	serviceAccountID := envLoader.GetAndLogEnvVar(envPrefix+"SERVICE_ACCOUNT_ID", false, false)
	privKeyRaw := envLoader.GetAndLogEnvVar(envPrefix+"PRIV_KEY", true, true)
	privKeyRaw = strings.ReplaceAll(privKeyRaw, "\\n", "\n")
	privKey, err := keys.NewPrivKey(keys.PS256, privKeyRaw)
	if err != nil {
		logger.Errorf("Error parsing priv key: %v", err)
	} else if serviceAccountID == "" {
		logger.Errorf("Missing service account id")
	} else {
		signatureAuth, err := sigauth.NewSignatureAuth(privKey, serviceRegManager, false, false)
		if err != nil {
			logger.Fatalf("Error initializing signature auth: %v", err)
		}

		serviceAccountLoader, err := authservice.NewRemoteServiceAccountLoader(&authService, serviceAccountID, signatureAuth)
		if err != nil {
			logger.Fatalf("Error initializing remote service account loader: %v", err)
		}

		serviceAccountManager, err = authservice.NewServiceAccountManager(&authService, serviceAccountLoader)
		if err != nil {
			logger.Fatalf("Error initializing service account manager: %v", err)
		}
	}
	config := &model.Config{
		MongoDBAuth:       mongoDBAuth,
		MongoDBName:       mongoDBName,
		MongoTimeout:      mongoTimeout,
		InternalAPIKey:    internalAPIKey,
		CoreBBHost:        coreBBHost,
		PollServiceURL:    serviceURL,
		UiucOrgID:         uiucOrgID,
		GroupsHost:        groupsBBHost,
		NotificationsHost: notificationsBBHost,
	}

	storageAdapter := storage.NewStorageAdapter(config, logger)
	err = storageAdapter.Start()
	if err != nil {
		log.Fatal("Cannot start the mongoDB adapter - " + err.Error())
	} else {
		log.Printf("Storage started")
	}

	//notifications BB adapter
	appID := envLoader.GetAndLogEnvVar(envPrefix+"APP_ID", true, true)
	orgID := envLoader.GetAndLogEnvVar(envPrefix+"ORG_ID", true, true)
	notificationsBBAdapter := notifications.NewNotificationsAdapter(notificationsBBHost, internalAPIKey, appID, orgID)

	groupsAdapter := groups.NewGroupsAdapter(config)

	defaultCacheExpirationSeconds := envLoader.GetAndLogEnvVar("DEFAULT_CACHE_EXPIRATION_SECONDS", false, false)
	cacheAdapter := cacheadapter.NewCacheAdapter(defaultCacheExpirationSeconds)

	//core adapter
	coreAdapter := corebb.NewCoreAdapter(coreBBHost, orgID, appID, serviceAccountManager)

	// application
	application := core.NewApplication(Version, Build, storageAdapter, cacheAdapter, notificationsBBAdapter,
		groupsAdapter, serviceID, coreAdapter, logger)
	application.Start()

	var corsAllowedHeaders []string
	var corsAllowedOrigins []string
	corsAllowedHeadersStr := envLoader.GetAndLogEnvVar(envPrefix+"CORS_ALLOWED_HEADERS", false, true)
	if corsAllowedHeadersStr != "" {
		corsAllowedHeaders = strings.Split(corsAllowedHeadersStr, ",")
	}
	corsAllowedOriginsStr := envLoader.GetAndLogEnvVar(envPrefix+"CORS_ALLOWED_ORIGINS", false, true)
	if corsAllowedOriginsStr != "" {
		corsAllowedOrigins = strings.Split(corsAllowedOriginsStr, ",")
	}

	webAdapter := driver.NewWebAdapter(host, port, application, config, serviceRegManager, corsAllowedOrigins, corsAllowedHeaders, logger)

	webAdapter.Start()
}
