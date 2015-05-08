// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ae

import (
	//"time"

	"appengine"
	"appengine/datastore"

	"github.com/okinotes/okinotes"
)

type repository struct {
	c appengine.Context
}

func userKey(c appengine.Context, userName string) *datastore.Key {
	return datastore.NewKey(c, "User", userName, 0, nil)
}
func pageKey(c appengine.Context, userName, pageName string) *datastore.Key {
	return datastore.NewKey(c, "Page", pageName, 0, userKey(c, userName))
}
func itemKey(c appengine.Context, userName, pageName, itemID string) *datastore.Key {
	return datastore.NewKey(c, "Item", itemID, 0, pageKey(c, userName, pageName))
}
func templateKey(c appengine.Context, templateID string) *datastore.Key {
	return datastore.NewKey(c, "Template", templateID, 0, nil)
}
func imageKey(c appengine.Context, userName string, imgID string) *datastore.Key {
	return datastore.NewKey(c, "UploadInfo", imgID, 0, userKey(c, userName))
}
func usageKey(c appengine.Context, userName string, pageName string, imgID string) *datastore.Key {
	return datastore.NewKey(c, "Usage", imgID, 0, pageKey(c, userName, pageName))
}

func (repo repository) RunInTransaction(f func(repo okinotes.Repository) error) error {

	err := datastore.RunInTransaction(repo.c, func(c appengine.Context) error {
		txRepo := repository{c}

		return f(txRepo)
	}, nil)

	return err
}
func (repo repository) GetPage(userName string, pageName string) (okinotes.Page, error) {

	var page okinotes.Page

	err := datastore.Get(repo.c, pageKey(repo.c, userName, pageName), &page)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return okinotes.Page{}, err
	}
	if err == datastore.ErrNoSuchEntity {
		return okinotes.Page{}, okinotes.NotInDatastoreError{"Page", pageName}
	}

	return page, nil
}
func (repo repository) StorePage(page okinotes.Page) error {
	_, err := datastore.Put(repo.c, pageKey(repo.c, page.UserName, page.Name), &page)
	return err
}
func (repo repository) DeletePage(userName, pageName string) error {
	return datastore.Delete(repo.c, pageKey(repo.c, userName, pageName))
}

type aePageQuery struct {
	c     appengine.Context
	q     *datastore.Query
	limit int
}

func (repo repository) NewPageQuery() okinotes.PageQuery {
	return &aePageQuery{
		repo.c,
		datastore.NewQuery("Page"),
		-1,
	}
}
func (q *aePageQuery) User(userName string) okinotes.PageQuery {
	q.q = q.q.Ancestor(userKey(q.c, userName))
	return q
}
func (q *aePageQuery) Filter(filterStr string, value interface{}) okinotes.PageQuery {
	q.q = q.q.Filter(filterStr, value)
	return q
}
func (q *aePageQuery) Order(fieldName string) okinotes.PageQuery {
	q.q = q.q.Order(fieldName)
	return q
}
func (q *aePageQuery) Limit(limit int) okinotes.PageQuery {
	q.q = q.q.Limit(limit + 1)
	q.limit = limit
	return q
}

func (q *aePageQuery) GetAll() ([]okinotes.Page, bool, error) { //TODO: use cursor string instead of bool
	var pages []okinotes.Page

	_, err := q.q.GetAll(q.c, &pages)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, false, err
	}
	if q.limit > 0 && len(pages) > q.limit {
		return pages[:q.limit], true, nil
	}
	return pages, false, nil
}

func (repo repository) GetItemsFromPage(userName, pageName string, limit int) ([]okinotes.Item, error) {
	var items []okinotes.Item

	_, err := datastore.NewQuery("Item").Ancestor(pageKey(repo.c, userName, pageName)).Order("-LastModificationDate").Limit(limit).GetAll(repo.c, &items)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	return items, nil
}
func (repo repository) FindItem(userName string, pageName string, itemID string) (bool, error) {

	item := okinotes.Item{}
	err := datastore.Get(repo.c, itemKey(repo.c, userName, pageName, itemID), &item)
	if err == datastore.ErrNoSuchEntity {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, err
}
func (repo repository) GetItem(userName string, pageName string, itemID string) (okinotes.Item, error) {

	item := okinotes.Item{}
	err := datastore.Get(repo.c, itemKey(repo.c, userName, pageName, itemID), &item)
	if err == datastore.ErrNoSuchEntity {
		return okinotes.Item{}, okinotes.NotInDatastoreError{"Item", itemID}
	}
	if err != nil {
		return okinotes.Item{}, err
	}

	return item, err
}
func (repo repository) StoreItem(userName string, pageName string, i okinotes.Item) error {
	_, err := datastore.Put(repo.c, itemKey(repo.c, userName, pageName, i.ID), &i)
	return err
}
func (repo repository) DeleteItem(userName, pageName, itemID string) error {
	return datastore.Delete(repo.c, itemKey(repo.c, userName, pageName, itemID))
}
func (repo repository) DeleteItemsFromPage(userName, pageName string) error {

	keys, err := datastore.NewQuery("Item").Ancestor(pageKey(repo.c, userName, pageName)).KeysOnly().GetAll(repo.c, nil)
	if err != nil {
		return err
	}

	return datastore.DeleteMulti(repo.c, keys)
}

func (repo repository) FindUser(userName string) (bool, error) {

	user := okinotes.User{}
	err := datastore.Get(repo.c, userKey(repo.c, userName), &user)
	if err == datastore.ErrNoSuchEntity {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, err
}
func (repo repository) GetUser(ident okinotes.Ident) (okinotes.User, error) {

	identity, err := repo.GetIdentity(ident)
	if err != nil {
		return okinotes.User{}, err
	}
	if len(identity.Identity) == 0 {
		return okinotes.User{}, nil
	}

	user := okinotes.User{}
	err = datastore.Get(repo.c, userKey(repo.c, identity.UserName), &user)
	if err == datastore.ErrNoSuchEntity {
		return okinotes.User{}, nil
	}
	if err != nil {
		return okinotes.User{}, err
	}

	return user, nil
}
func (repo repository) GetIdentity(ident okinotes.Ident) (okinotes.Identity, error) {

	var identities []okinotes.Identity

	_, err := datastore.NewQuery("Identity").Filter("Provider =", ident.Provider).Filter("Identity =", ident.Identity).Limit(1).GetAll(repo.c, &identities)
	if err != nil {
		return okinotes.Identity{}, err
	}

	if len(identities) == 0 {
		return okinotes.Identity{}, okinotes.NotInDatastoreError{"Identity", ident.Identity}
	}

	return identities[0], nil
}
func (repo repository) StoreUser(user okinotes.User) error {
	_, err := datastore.Put(repo.c, userKey(repo.c, user.Name), &user)
	return err
}
func (repo repository) StoreIdentity(identity okinotes.Identity) error {
	_, err := datastore.Put(repo.c, datastore.NewIncompleteKey(repo.c, "Identity", userKey(repo.c, identity.UserName)), &identity)
	return err

}

func (repo repository) StoreImage(img okinotes.UploadInfo, userName string) error {
	_, err := datastore.Put(repo.c, imageKey(repo.c, userName, img.Key), &img)
	return err
}
func (repo repository) RenameImage(name string, imgID string, userName string) error {

	err := datastore.RunInTransaction(repo.c, func(c appengine.Context) error {
		k := imageKey(repo.c, userName, imgID)
		var oldImg okinotes.UploadInfo
		err := datastore.Get(repo.c, k, &oldImg)
		if err != nil {
			return err
		}

		if oldImg.Filename == name {
			return nil
		}
		oldImg.Filename = name

		_, err = datastore.Put(repo.c, k, &oldImg)

		return err
	}, nil)

	return err
}
func (repo repository) GetImages(userName string, limit int) ([]okinotes.UploadInfo, error) {

	var imgs []okinotes.UploadInfo

	_, err := datastore.NewQuery("UploadInfo").Ancestor(userKey(repo.c, userName)).Order("Filename").Limit(limit).GetAll(repo.c, &imgs)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	return imgs, nil
}
func (repo repository) DeleteImage(imgID string, userName string) error {
	return datastore.Delete(repo.c, imageKey(repo.c, userName, imgID))
}

func (repo repository) IsUsed(imgID string) (bool, error) {

	_, err := datastore.NewQuery("Usage").Filter("UploadInfoKey =", imgID).Limit(1).KeysOnly().GetAll(repo.c, nil)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return false, err
	}

	return (err == nil), nil
}

func (repo repository) StoreUsage(userName string, pageName string, imgID string) error {
	usage := okinotes.Usage{UploadInfoKey: imgID}
	_, err := datastore.Put(repo.c, usageKey(repo.c, userName, pageName, imgID), &usage)
	return err
}

func (repo repository) DeleteUsages(userName string, pageName string) error {

	keys, err := datastore.NewQuery("Usage").Ancestor(pageKey(repo.c, userName, pageName)).KeysOnly().GetAll(repo.c, nil)
	if err != nil {
		return err
	}

	return datastore.DeleteMulti(repo.c, keys)
}

func (repo repository) GetTemplate(templateID string) (okinotes.Template, error) {
	var template okinotes.Template

	err := datastore.Get(repo.c, templateKey(repo.c, templateID), &template)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return okinotes.Template{}, err
	}
	if err == datastore.ErrNoSuchEntity {
		return okinotes.Template{}, okinotes.NotInDatastoreError{"Template", templateID}
	}

	return template, nil
}
func (repo repository) GetAllTemplates() ([]okinotes.Template, error) {
	var templates []okinotes.Template

	_, err := datastore.NewQuery("Template").Order("Name").GetAll(repo.c, &templates)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	return templates, nil
}

func (repo repository) StoreTemplate(tpl okinotes.Template, generateID func() string) (string, error) {
	err := datastore.RunInTransaction(repo.c, func(c appengine.Context) error {

		//Generate a new item id if not set
		if len(tpl.ID) == 0 {
			for {
				tpl.ID = generateID()
				//Check for uniqueness
				k := templateKey(c, tpl.ID)
				var oldTemplate okinotes.Template
				err := datastore.Get(repo.c, k, &oldTemplate)
				if err != nil && err != datastore.ErrNoSuchEntity {
					return err
				}
				if err == datastore.ErrNoSuchEntity {
					break
				}
			}
		}

		k := templateKey(c, tpl.ID)
		var oldTemplate okinotes.Template
		err := datastore.Get(repo.c, k, &oldTemplate)
		if err != nil && err != datastore.ErrNoSuchEntity {
			return err
		}

		if err == datastore.ErrNoSuchEntity {
			tpl.CreationDate = tpl.LastModificationDate
		} else {
			tpl.CreationDate = oldTemplate.CreationDate
		}

		_, err = datastore.Put(c, k, &tpl)

		return err
	}, nil)

	return tpl.ID, err
}

func (repo repository) DeleteTemplate(templateID string) error {
	return datastore.Delete(repo.c, templateKey(repo.c, templateID))
}
