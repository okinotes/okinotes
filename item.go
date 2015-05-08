// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"html/template"
	"time"
)

//Item is an element of a page.
//It belongs to his parent page.
type Item struct {
	ID string `json:"id"`

	CreationDate         time.Time `json:"creationDate"`
	LastModificationDate time.Time `json:"lastModificationDate"`

	Kind        string        `json:"kind"`
	Title       string        `json:"title"`
	Content     string        `datastore:",noindex"  json:"content"`
	HTMLContent template.HTML `datastore:",noindex"  json:"htmlContent"`
	Source      string        `json:"source"`
	URL         string        `json:"url"`

	Tags TagList `json:"tags"`
}

//Compute a markdown representation of the Item
func (i Item) String() string {
	//TODO: return should depends on i.Kind
	return i.Title
}
