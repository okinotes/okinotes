// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ae

import (
	"appengine"
	"appengine/user"

	"github.com/okinotes/okinotes"
)

type userInteractor struct {
	c appengine.Context
}

func (i userInteractor) CurrentIdentity() (okinotes.Ident, error) {

	u := user.Current(i.c)
	if u == nil {
		return okinotes.Ident{}, nil
	}

	return okinotes.Ident{"Google", u.ID}, nil
}

func (i userInteractor) CurrentUserIsAdmin() bool {
	u := user.Current(i.c)
	if u == nil {
		return false
	}

	return u.Admin
}

func (i userInteractor) LoginURL(destURL string) (string, error) {
	return user.LoginURL(i.c, destURL)
}

func (i userInteractor) LogoutURL(destURL string) (string, error) {
	return user.LogoutURL(i.c, destURL)
}
