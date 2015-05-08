// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"fmt"
)

//PageQuery allows querying multiple pages using conditions and ordering
type PageQuery interface {
	User(userName string) PageQuery
	Filter(filterStr string, value interface{}) PageQuery
	Order(fieldName string) PageQuery
	Limit(limit int) PageQuery

	GetAll() ([]Page, bool, error)
}

//Repository is the interface allowing usage of any data store for Pages, Items and all other data
type Repository interface {
	RunInTransaction(func(repo Repository) error) error

	GetPage(userName string, pageName string) (Page, error)
	NewPageQuery() PageQuery
	StorePage(page Page) error
	DeletePage(userName string, pageName string) error

	GetItemsFromPage(userName string, pageName string, limit int) ([]Item, error)
	DeleteItemsFromPage(userName string, pageName string) error
	FindItem(userName string, pageName string, itemID string) (bool, error)
	GetItem(userName string, pageName string, itemID string) (Item, error)
	StoreItem(userName string, pageName string, i Item) error
	DeleteItem(userName string, pageName string, itemID string) error

	FindUser(userName string) (bool, error)
	GetUser(ident Ident) (User, error)
	GetIdentity(ident Ident) (Identity, error)
	StoreUser(user User) error
	StoreIdentity(identity Identity) error

	GetImages(userName string, limit int) ([]UploadInfo, error)
	StoreImage(img UploadInfo, userName string) error
	RenameImage(name string, imgID string, userName string) error
	DeleteImage(imgID string, userName string) error

	IsUsed(imgID string) (bool, error)
	StoreUsage(userName string, pageName string, imgID string) error
	DeleteUsages(userName string, pageName string) error

	GetTemplate(templateID string) (Template, error)
	GetAllTemplates() ([]Template, error)
	StoreTemplate(tpl Template, generateID func() string) (string, error)
	DeleteTemplate(templateID string) error
}

//NotInDatastoreError represents an error on data not in datastore
type NotInDatastoreError struct {
	Type string
	ID   string
}

func (err NotInDatastoreError) Error() string {
	return fmt.Sprintf("%s '%s' not found.", err.Type, err.ID)
}
