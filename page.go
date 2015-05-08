// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"time"
)

//A Policy defines the accessibility of content
type Policy string

const (
	//PolicyPRIVATE means that only authorized user may access the data
	PolicyPRIVATE Policy = "PRIVATE"
	//PolicyPUBLIC means that anonymous user may access the data
	PolicyPUBLIC Policy = "PUBLIC"
)

//Page represents a collection of items. It belongs to a user
type Page struct {
	UserName             string    `json:"userName"`
	Name                 string    `json:"name"`
	CreationDate         time.Time `json:"creationDate"`
	LastModificationDate time.Time `json:"lastModificationDate"`
	Title                string    `json:"title"`
	ContentLicense       string    `json:"contentLicense"`
	Policy               Policy    `json:"policy"`
	TemplateID           string    `json:"templateID"`
	Tags                 TagList   `json:"tags"`
}

//Usage represents the characteristics of the usage of a
//ressource in a page.
type Usage struct {
	UploadInfoKey string
}
