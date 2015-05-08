// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"errors"
	"fmt"
	"testing"
)

type testAppFactory struct {
	CurrentUserID      string
	CurrentUserIsAdmin bool
}

func (f testAppFactory) CreateApp() (App, error) {
	repo := testRepository{
		pages: make(map[string]map[string]Page),
		items: make(map[string]Item),
	}

	return NewApp(
		&repo,
		&testUserInteractor{f.CurrentUserID, f.CurrentUserIsAdmin},
		&testLogInteractor{},
		nil,
	), nil
}

type testRepository struct {
	pages map[string]map[string]Page
	items map[string]Item
	users []User
}

func (repo *testRepository) RunInTransaction(f func(repo Repository) error) error {
	backup := *repo

	err := f(repo)

	if err != nil {
		*repo = backup
	}
	return err
}

func (repo *testRepository) GetPage(userName string, pageName string) (Page, error) {

	pages, found := repo.pages[userName]

	if !found {
		return Page{}, NotInDatastoreError{"Page", pageName}
	}

	page, found := pages[pageName]

	if !found {
		return Page{}, NotInDatastoreError{"Page", pageName}
	}

	return page, nil
}
func (repo *testRepository) NewPageQuery() PageQuery {
	return nil
}
func (repo *testRepository) StorePage(page Page) error {
	repo.pages[page.UserName][page.Name] = page

	return nil
}
func (repo *testRepository) DeletePage(userName string, pageName string) error {
	return errors.New("Not implemented")
}

func (repo *testRepository) GetItemsFromPage(userName string, pageName string, limit int) ([]Item, error) {
	return nil, errors.New("Not implemented")
}
func (repo *testRepository) DeleteItemsFromPage(userName string, pageName string) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) FindItem(userName string, pageName string, itemID string) (bool, error) {
	return false, errors.New("Not implemented")
}
func (repo *testRepository) GetItem(userName string, pageName string, itemID string) (Item, error) {
	return Item{}, errors.New("Not implemented")
}
func (repo *testRepository) StoreItem(userName string, pageName string, i Item) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) DeleteItem(userName string, pageName string, itemID string) error {
	return errors.New("Not implemented")
}

func (repo *testRepository) FindUser(userName string) (bool, error) {
	return false, errors.New("Not implemented")
}
func (repo *testRepository) GetUser(ident Ident) (User, error) {
	return User{}, errors.New("Not implemented")
}
func (repo *testRepository) GetIdentity(ident Ident) (Identity, error) {
	return Identity{}, errors.New("Not implemented")
}
func (repo *testRepository) StoreUser(user User) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) StoreIdentity(identity Identity) error {
	return errors.New("Not implemented")
}

func (repo *testRepository) GetImages(userName string, limit int) ([]UploadInfo, error) {
	return nil, errors.New("Not implemented")
}
func (repo *testRepository) StoreImage(img UploadInfo, userName string) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) RenameImage(name string, imgID string, userName string) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) DeleteImage(imgID string, userName string) error {
	return errors.New("Not implemented")
}

func (repo *testRepository) IsUsed(imgID string) (bool, error) {
	return false, errors.New("Not implemented")
}
func (repo *testRepository) StoreUsage(userName string, pageName string, imgID string) error {
	return errors.New("Not implemented")
}
func (repo *testRepository) DeleteUsages(userName string, pageName string) error {
	return errors.New("Not implemented")
}

func (repo *testRepository) GetTemplate(templateID string) (Template, error) {
	return Template{}, errors.New("Not implemented")
}
func (repo *testRepository) GetAllTemplates() ([]Template, error) {
	return nil, errors.New("Not implemented")
}
func (repo *testRepository) StoreTemplate(tpl Template, generateID func() string) (string, error) {
	return "", errors.New("Not implemented")
}
func (repo *testRepository) DeleteTemplate(templateID string) error {
	return errors.New("Not implemented")
}

type testUserInteractor struct {
	currentUserID      string
	currentUserIsAdmin bool
}

func (i *testUserInteractor) CurrentIdentity() (Ident, error) {
	return Ident{}, errors.New("Not implemented")
}
func (i *testUserInteractor) CurrentUserIsAdmin() bool {
	return i.currentUserIsAdmin
}

func (i *testUserInteractor) LoginURL(destURL string) (string, error) {
	return "", errors.New("Not implemented")
}
func (i *testUserInteractor) LogoutURL(destURL string) (string, error) {
	return "", errors.New("Not implemented")
}

type testLogInteractor struct {
}

func (logger *testLogInteractor) Debugf(format string, args ...interface{}) {
	fmt.Printf("DEBUG:    "+format, args)
}
func (logger *testLogInteractor) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO:     "+format, args)
}
func (logger *testLogInteractor) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING:  "+format, args)
}
func (logger *testLogInteractor) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR:    "+format, args)
}
func (logger *testLogInteractor) Criticalf(format string, args ...interface{}) {
	fmt.Printf("CRITICAL: "+format, args)
}

func TestListPages(t *testing.T) {

	//Get empty list
	{
		f := testAppFactory{"user01", false}
		app, _ := f.CreateApp()
		var in []string
		out, _, err := app.ListOwnedPages(100)

		if err != nil {
			t.Error(err)
		}
		if len(out) != 0 {
			t.Errorf("GetPages(%s) = %s, wanted empty list", in, out)
		}
	}
	//Admin
	{
		f := testAppFactory{"user01", true}
		app, _ := f.CreateApp()
		in := "user02"
		//TODO
		out, _, err := app.ListOwnedPages(100)

		if err != nil {
			t.Error(err)
		}
		if len(out) != 0 {
			t.Errorf("GetPages(%s) = %s, wanted empty list", in, out)
		}
	}

	//Get existing item
	{
		f := testAppFactory{"user01", false}
		app, _ := f.CreateApp()
		app.CreatePage(Page{Title: "page 01"})
		app.CreatePage(Page{Title: "page 02"})
		app.CreatePage(Page{Title: "page 03"})
		app.CreatePage(Page{Title: "page 04"})
		app.CreatePage(Page{Title: "page 05"})

		out, _, err := app.ListOwnedPages(100)

		if err != nil {
			t.Error(err)
		}
		if len(out) != 5 {
			t.Errorf("GetPages(%s) = %s, wanted list with all pages", "user01", out)
		}
	}

	//Get existing with limit
	{
		f := testAppFactory{"user01", false}
		app, _ := f.CreateApp()
		err := app.CreatePage(Page{Title: "page 01"})
		if err != nil {
			t.Error(err)
		}
		app.CreatePage(Page{Title: "page 02"})
		app.CreatePage(Page{Title: "page 03"})
		app.CreatePage(Page{Title: "page 04"})
		app.CreatePage(Page{Title: "page 05"})

		limit := 3
		out, _, err := app.ListOwnedPages(limit)

		if err != nil {
			t.Error(err)
		}
		if len(out) != limit {
			t.Errorf("GetPages(%s) = %s, wanted limit to %d, app=%+v", "user01", out, limit, app.repository)
		}
	}

}
