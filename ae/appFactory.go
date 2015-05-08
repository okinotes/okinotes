// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ae

import (
	"net/http"

	"appengine"

	"github.com/okinotes/okinotes"
)

//AppFactory is a factory for okinotes.App, based on the services provided by appengine
type AppFactory struct{}

//CreateApp creates a new okinotes.App, based on the services provided by appengine
func (f AppFactory) CreateApp(r *http.Request) (okinotes.App, error) {
	c := appengine.NewContext(r)

	repository := repository{c}
	userInteractor := userInteractor{c}
	logInteractor := c
	uploadInteractor := uploadInteractor{c}

	app := okinotes.NewApp(repository, userInteractor, logInteractor, uploadInteractor)

	return app, nil
}
