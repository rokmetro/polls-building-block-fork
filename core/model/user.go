package model

import (
	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth/tokenauth"
)

// User auth wrapper
type User struct {
	Token  string
	Claims tokenauth.Claims
}

// UserRef reference for a concrete user which is member of a group
type UserRef struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
} // @name MemberRecipient

// Sender Wraps sender type and user ref
type Sender struct {
	Type string   `json:"type" bson:"type"` // user or system
	User *UserRef `json:"user,omitempty" bson:"user,omitempty"`
} // @name Sender

// DeletedUserData represents a user-deleted
type DeletedUserData struct {
	AppID       string              `json:"app_id"`
	Memberships []DeletedMembership `json:"memberships"`
	OrgID       string              `json:"org_id"`
}

// DeletedMembership defines model for DeletedMembership.
type DeletedMembership struct {
	AccountID string                  `json:"account_id"`
	Context   *map[string]interface{} `json:"context,omitempty"`
}

// UserDataResponse wraps polls user data
type UserDataResponse struct {
	Poll           []Poll           `json:"my_polls"`
	Surveys        []Survey         `json:"my_surveys"`
	SurveyResponse []SurveyResponse `json:"participated_surveys"`
} //@name UserDataResponse
