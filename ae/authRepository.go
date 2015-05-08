// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ae

// import (
// "time"

// "errors"

// "appengine"
// "appengine/datastore"

// "github.com/xeonx/wizy/app"
// "github.com/xeonx/wizy/okinotes"
// )

// type authRepository struct {
// c appengine.Context
// }

// func permissionKey(c appengine.Context, userId string, pageId string) *datastore.Key {
// return datastore.NewKey(c, "Permission", userId+"_"+pageId, 0, pageKey(c, pageId))
// }

// func (repo authRepository) IsRegistered(userId string) (bool, error) {
// if len(userId) == 0 {
// return false, nil
// }

// user := okinotes.User{}
// err := datastore.Get(repo.c, userKey(repo.c, userId), &user)
// if err == datastore.ErrNoSuchEntity {
// return false, nil
// }
// if err != nil {
// return false, err
// }

// return true, nil
// }
// func (repo authRepository) StoreUser(user okinotes.User) error {
// if len(user.Id) == 0 {
// return errors.New("Undefined user id. Unable to store user.")
// }

// _, err := datastore.Put(repo.c, userKey(repo.c, user.Id), &user)
// if err != nil {
// return err
// }

// return nil
// }
// func (repo authRepository) GetUserPermissions(userId string, limit int, ownedOnly bool) ([]okinotes.Permission, bool, error) {

// var permissions []okinotes.Permission

// q := datastore.NewQuery("Permission").Filter("UserId =", userId).Order("-PageLastUpdateDate")
// if ownedOnly {
// q = q.Filter("Role =", okinotes.Role_Owner)
// }

// q = q.Limit(limit + 1) //Limit is set to requested value +1 in order to know if there is more data available in the datatstore

// _, err := q.GetAll(repo.c, &permissions)
// if err != nil && err != datastore.ErrNoSuchEntity {
// return nil, false, err
// }

// if len(permissions) > limit {
// return permissions[:limit], true, nil
// }

// return permissions, false, nil
// }
// func (repo authRepository) GetPermission(userId, pageId string) (okinotes.Permission, error) {
// var permission okinotes.Permission

// err := datastore.Get(repo.c, permissionKey(repo.c, userId, pageId), &permission)
// if err != nil && err != datastore.ErrNoSuchEntity {
// return okinotes.Permission{}, err
// }

// if err == datastore.ErrNoSuchEntity {

// //Check for page existence
// var page okinotes.Page
// err = datastore.Get(repo.c, pageKey(repo.c, pageId), &page)
// if err == datastore.ErrNoSuchEntity {
// return okinotes.Permission{}, app.NotInDatastoreError{"Page", pageId}
// }

// if page.Policy == okinotes.PolicyPUBLIC {
// permission.UserId = ""
// permission.PageId = page.Id
// permission.CreationDate = page.LastModificationDate
// permission.LastModificationDate = page.LastModificationDate
// permission.PageLastUpdateDate = page.LastModificationDate
// permission.Role = okinotes.Role_User
// return permission, nil
// }

// //Check for user existence
// var user okinotes.User
// err = datastore.Get(repo.c, userKey(repo.c, userId), &user)
// if err == datastore.ErrNoSuchEntity {
// return okinotes.Permission{}, app.NotInDatastoreError{"User", userId}
// }

// return okinotes.Permission{}, app.NotInDatastoreError{"Permission", userId + "_" + pageId}
// }

// return permission, nil
// }
// func (repo authRepository) StorePermission(p okinotes.Permission) error {

// return datastore.RunInTransaction(repo.c, func(c appengine.Context) error {
// k := permissionKey(c, p.UserId, p.PageId)
// var oldPermission okinotes.Permission

// err := datastore.Get(repo.c, k, &oldPermission)
// if err != nil && err != datastore.ErrNoSuchEntity {
// return err
// }

// if err == datastore.ErrNoSuchEntity {
// p.CreationDate = p.LastModificationDate
// } else {
// p.CreationDate = oldPermission.CreationDate
// }

// _, err = datastore.Put(c, k, &p)

// return err
// }, nil)
// }
// func (repo authRepository) DeletePermission(p okinotes.Permission) error {
// return datastore.Delete(repo.c, permissionKey(repo.c, p.UserId, p.PageId))
// }
// func (repo authRepository) DeletePermissionFromPage(pageId string) error {

// keys, err := datastore.NewQuery("Permission").Ancestor(pageKey(repo.c, pageId)).KeysOnly().GetAll(repo.c, nil)
// if err != nil {
// return err
// }

// return datastore.DeleteMulti(repo.c, keys)
// }
// func (repo authRepository) UpdatePageLastUpdateDate(pageId string, lastModificationDate time.Time) error {

// return datastore.RunInTransaction(repo.c, func(c appengine.Context) error {

// q := datastore.NewQuery("Permission").Ancestor(pageKey(c, pageId))
// t := q.Run(c)
// for {
// var p okinotes.Permission
// k, err := t.Next(&p)
// if err == datastore.Done {
// break // No further entities match the query.
// }
// if err != nil {
// return err
// }

// p.PageLastUpdateDate = lastModificationDate

// _, err = datastore.Put(c, k, &p)
// if err != nil {
// return err
// }
// }

// return nil
// }, nil)
// }
