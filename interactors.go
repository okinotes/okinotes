// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"net/http"
)

//UserInteractor allows interactions with the User connected to the application
type UserInteractor interface {
	CurrentIdentity() (Ident, error)
	CurrentUserIsAdmin() bool

	LoginURL(destURL string) (string, error)
	LogoutURL(destURL string) (string, error)
}

//UploadInteractor allows uploading and serving images
type UploadInteractor interface {
	UploadURL(destURL string, maxUploadBytes int64) (string, error)
	UploadInfo(req *http.Request, name string) (UploadInfo, error)
	ImageURL(key string, secure bool, size int) (string, error)
	Delete(key string) error
}

//LogInteractor allows logging of application messages
type LogInteractor interface {
	// Debugf formats its arguments according to the format, analogous to fmt.Printf,
	// and records the text as a log message at Debug level.
	Debugf(format string, args ...interface{})

	// Infof is like Debugf, but at Info level.
	Infof(format string, args ...interface{})

	// Warningf is like Debugf, but at Warning level.
	Warningf(format string, args ...interface{})

	// Errorf is like Debugf, but at Error level.
	Errorf(format string, args ...interface{})

	// Criticalf is like Debugf, but at Critical level.
	Criticalf(format string, args ...interface{})
}
