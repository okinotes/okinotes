// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"net/http"
)

type handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}
type redirectHandler struct {
	Target string
}

func (c redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	http.Redirect(w, r, c.Target, http.StatusSeeOther)
	return nil
}

type templateHandler struct {
	Template string
	Data     interface{}
}

func (c templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return allTemplates.ExecuteTemplate(w, c.Template, c.Data)
}

type marshalHandler struct {
	Marshal       func(v interface{}) ([]byte, error)
	Data          interface{}
	MimeTypeValue string
	FileNameValue string
}

func (c marshalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {

	if len(c.MimeTypeValue) > 0 {
		w.Header().Set("Content-Type", c.MimeTypeValue)
	}

	if len(c.FileNameValue) > 0 {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+c.FileNameValue+"\";")
	}

	b, err := c.Marshal(c.Data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		return err
	}
	return nil
}
