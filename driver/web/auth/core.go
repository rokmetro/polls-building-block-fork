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
	"log"
	"net/http"
	"polls/core"
	"polls/core/model"

	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth"
	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth/authorization"
	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth/tokenauth"
)

// CoreAuth implementation
type CoreAuth struct {
	app       *core.Application
	tokenAuth *tokenauth.TokenAuth
}

// Check checks the request contains a valid Core access token
func (ca CoreAuth) Check(r *http.Request) (bool, *model.User) {

	claims, err := ca.tokenAuth.CheckRequestToken(r)
	if err != nil {
		log.Printf("%s", err)
		return false, nil
	}

	token, _ := tokenauth.GetAccessToken(r)

	return true, &model.User{
		Token:  token,
		Claims: *claims,
	}
}

// CheckWithAuthorization checks the request contains a valid Core access token + authorization
func (ca CoreAuth) CheckWithAuthorization(r *http.Request) (bool, bool, *model.User) {

	claims, err := ca.tokenAuth.CheckRequestToken(r)
	if err != nil {
		log.Printf("%s", err)
		return false, false, nil
	}

	err = ca.tokenAuth.AuthorizeRequestPermissions(claims, r)
	if err != nil {
		log.Printf("%s", err)
		return true, false, nil
	}

	token, _ := tokenauth.GetAccessToken(r)

	return true, true, &model.User{
		Token:  token,
		Claims: *claims,
	}
}

// NewCoreAuth creates new CoreAuth
func NewCoreAuth(app *core.Application, serviceRegManager *auth.ServiceRegManager) *CoreAuth {
	permissionAuth := authorization.NewCasbinStringAuthorization("driver/web/authorization_policy.csv")
	tokenAuth, err := tokenauth.NewTokenAuth(true, serviceRegManager, permissionAuth, nil)
	if err != nil {
		log.Fatalf("error on new core auth")
		return nil
	}

	//standartHandler := tokenauth.NewStandardHandler(*tokenAuth, nil)

	auth := CoreAuth{app: app, tokenAuth: tokenAuth}
	return &auth
}
