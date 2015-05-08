// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

//UserKind represents the kind of a user
type UserKind string

const (
	//UserKindUSER are users having an identity
	UserKindUSER UserKind = "USER"
	//UserKindORG represents a group of users
	UserKindORG  UserKind = "ORG"
)

//User represents someone interacting with notes
type User struct {
	Name     string //Uniquely identify the user
	Kind     UserKind
	FullName string
}

//Ident is an identity from a third party provider
type Ident struct {
	Provider string
	Identity string
}

//Identity is the authentication of a user (the link between an external Ident and a User)
type Identity struct {
	Ident

	UserName string
}

//Membership indicates the membership of a user into an organisation.
type Membership struct {
	UserName string
	OrgName  string

	IsOrgAdmin bool
}

//Compute a markdown representation of the User
func (u User) String() string {
	if len(u.Name) > 0 {
		return u.Name
	}

	return "*Anonymous*"
}
