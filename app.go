// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//Package okinotes contains all the model and business layers for the okinotes application.
package okinotes

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"time"
)

var (
	//ErrFirstUserConnection is the errors returned when connecting
	//for the first time with an identity.
	ErrFirstUserConnection = errors.New("User not created")
)

//AppFactory represents the objects able to create an App
type AppFactory interface {
	CreateApp(r *http.Request) (App, error)
}

//App is the main application.
type App struct {
	repository       Repository
	userInteractor   UserInteractor
	logInteractor    LogInteractor
	uploadInteractor UploadInteractor
}

//NewApp creates a new App using the given services
func NewApp(r Repository, u UserInteractor, l LogInteractor, up UploadInteractor) App {
	return App{
		repository:       r,
		userInteractor:   u,
		logInteractor:    l,
		uploadInteractor: up,
	}
}

//CurrentUserName returns the name of the current user.
//Returns an empty string if not logged in.
func (app App) CurrentUserName() string {
	identity, err := app.CurrentIdentity()
	if err != nil {
		return ""
	}
	return identity.UserName
}

//CurrentUserIsAdmin returns true if the current user is an administrator
func (app App) CurrentUserIsAdmin() bool {
	return app.userInteractor.CurrentUserIsAdmin()
}

//CurrentUser returns the current user
func (app App) CurrentUser() (User, error) {
	ident, err := app.userInteractor.CurrentIdentity()
	if err != nil {
		app.logInteractor.Infof("CurrentUser error: %v", err)
		return User{}, err
	}

	u, err := app.repository.GetUser(ident)
	if err != nil {
		if _, notFound := err.(NotInDatastoreError); notFound {
			err = ErrFirstUserConnection
		}
		return User{}, err
	}

	return u, nil
}

//CurrentIdentity returns the Identity used to logged in.
func (app App) CurrentIdentity() (Identity, error) {
	ident, err := app.userInteractor.CurrentIdentity()
	if err != nil {
		app.logInteractor.Infof("CurrentIdentity error: %v", err)
		return Identity{}, err
	}
	app.logInteractor.Infof("ident.Identity: %s", ident.Identity)
	if len(ident.Identity) == 0 {
		return Identity{}, nil //Not logged in
	}

	id, err := app.repository.GetIdentity(ident)
	if err != nil {
		if _, notFound := err.(NotInDatastoreError); notFound {
			return Identity{Ident: ident}, ErrFirstUserConnection
		}
		app.logInteractor.Infof("CurrentIdentity->GetIdentity error: %v", err)
		return Identity{}, err
	}

	return id, nil
}

//LoginURL returns the URL to the log-in ressource
func (app App) LoginURL(destURL string) (string, error) {
	return app.userInteractor.LoginURL(destURL)
}

//LogoutURL returns the URL to the log-out ressource
func (app App) LogoutURL(destURL string) (string, error) {
	return app.userInteractor.LogoutURL(destURL)
}

//CreateUser creates a user
func (app App) CreateUser(ident Ident, userName string) error {

	//Check user name pattern
	ok, err := regexp.MatchString("[a-z0-9_\\-]{3,}", userName)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("The selected user name does not follow the required pattern.")
	}

	user := User{
		Name:     userName,
		Kind:     UserKindUSER,
		FullName: userName,
	}

	identity := Identity{ident, userName}

	return app.repository.RunInTransaction(func(repo Repository) error {

		exists, err := repo.FindUser(userName)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("The selected user name already exists")
		}

		err = repo.StoreUser(user)
		if err != nil {
			return err
		}
		err = repo.StoreIdentity(identity)
		if err != nil {
			return err
		}

		return nil
	})
}

//GetPage retrieve an existing single page
func (app App) GetPage(userName string, pageName string) (Page, error) {

	//Read the page in the datastore
	page, err := app.repository.GetPage(userName, pageName)
	if err != nil {
		app.logInteractor.Infof("GetPage error: %v", err)
		return Page{}, err
	}

	if page.Policy == PolicyPRIVATE && app.CurrentUserName() != userName {
		app.logInteractor.Infof("Not authorized to read the page")
		return Page{}, NotAuthorizedError{"Read page"}
	}

	return page, nil
}

//ListPublicPages returns the list of public pages for a given user
func (app App) ListPublicPages(limit int) ([]Page, bool, error) {

	//Get the list of public pages
	pages, bMore, err := app.repository.NewPageQuery().Filter("Policy =", PolicyPUBLIC).Order("-LastModificationDate").Limit(limit).GetAll()
	if err != nil {
		app.logInteractor.Errorf("ListPublicPages failed in GetAll(PUBLIC): %v", err)
		return nil, false, err
	}

	return pages, bMore, nil
}

//ListOwnedPages returns the list of pages owned by a given user
func (app App) ListOwnedPages(limit int) ([]Page, bool, error) {

	user, err := app.CurrentUser()
	if err != nil {
		app.logInteractor.Errorf("ListOwnedPages failed in CurrentUser: %v", err)
		return nil, false, err
	}

	//Get the list of public pages
	pages, bMore, err := app.repository.NewPageQuery().Filter("UserName=", user.Name).Order("-LastModificationDate").Limit(limit).GetAll()
	if err != nil {
		app.logInteractor.Errorf("ListOwnedPages failed in GetAll: %v", err)
		return nil, false, err
	}

	return pages, bMore, nil
}

//CreatePage create a new page for the given user
//Returns the id of the stored page.
func (app App) CreatePage(page Page) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 {
		return NotAuthorizedError{"Update page"}
	}
	page.UserName = currentUserName
	page.LastModificationDate = time.Now()
	page.CreationDate = page.LastModificationDate

	if len(page.Policy) == 0 {
		page.Policy = PolicyPRIVATE
	}

	err := app.repository.RunInTransaction(func(repo Repository) error {
		//Check for existence of user/page
		_, err := repo.GetPage(page.UserName, page.Name)
		if err == nil {
			return errors.New("Page already exists")
		}
		if _, notFound := err.(NotInDatastoreError); !notFound {
			return err
		}

		//Store
		return repo.StorePage(page)
	})

	return err
}

//UpdatePage updates a given page.
//The description of tags in the current template must be provided.
func (app App) UpdatePage(page Page, pageTags TagDescriptionList) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != page.UserName {
		return NotAuthorizedError{"Update page"}
	}

	page.LastModificationDate = time.Now()

	err := app.repository.RunInTransaction(func(repo Repository) error {
		//Check for existence of user/page
		oldPage, err := repo.GetPage(page.UserName, page.Name)
		if err != nil {
			return err
		}

		page.CreationDate = oldPage.CreationDate

		//Remove previous images usages
		err = repo.DeleteUsages(page.UserName, page.Name)
		if err != nil {
			return err
		}
		for _, t := range pageTags {
			if t.Kind == "imageId" {
				imgID := page.Tags.Tag(t.Key)

				err = repo.StoreUsage(page.UserName, page.Name, imgID)
				if err != nil {
					return err
				}
			}
		}

		//Store
		return repo.StorePage(page)
	})

	return err

}

//UpdateTemplate changes the template of page.
func (app App) UpdateTemplate(userName string, pageName string, newTemplateID string) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return NotAuthorizedError{"Update page"}
	}

	tNow := time.Now()

	err := app.repository.RunInTransaction(func(repo Repository) error {
		//Check for existence of user/page
		page, err := repo.GetPage(userName, pageName)
		if err != nil {
			return err
		}

		if page.TemplateID == newTemplateID {
			return nil
		}

		page.LastModificationDate = tNow
		page.TemplateID = newTemplateID

		//Store
		return repo.StorePage(page)
	})

	return err

}

//DeletePage removes permanently a page and all the associated content (items and permissions)
func (app App) DeletePage(userName, pageName string) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return NotAuthorizedError{"Delete page"}
	}

	//TODO: Use a transactional mechanism inn order to have them all succeed or all failed?

	//Delete items
	err := app.repository.DeleteItemsFromPage(userName, pageName)
	if err != nil {
		return err
	}

	//Delete page
	err = app.repository.DeletePage(userName, pageName)
	if err != nil {
		return err
	}

	return nil
}

//CreateItem stores an item
//Returns the stored item.
func (app App) CreateItem(userName, pageName string, i Item) (Item, error) {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return Item{}, NotAuthorizedError{"Store item"}
	}

	if len(i.Content) == 0 && len(i.URL) == 0 {
		return Item{}, fmt.Errorf("Empty item not allowed. Either content or URL must be provided: %#v", i)
	}

	//Compute HTML from markdown
	i.HTMLContent = template.HTML(markdownToHTML(i.Content))

	i.CreationDate = time.Now()
	i.LastModificationDate = i.CreationDate
	i.ID = generateID()

	//Stores the item
	err := app.repository.RunInTransaction(func(repo Repository) error {
		//Generate unique item ID
		for {
			found, err := repo.FindItem(userName, pageName, i.ID)
			if err != nil {
				return err
			}
			if !found {
				break
			}
			i.ID = generateID()
		}

		//Store
		return repo.StoreItem(userName, pageName, i)
	})
	if err != nil {
		return Item{}, err
	}

	{
		//Update LastModificationDate on Page
		err := app.repository.RunInTransaction(func(repo Repository) error {
			//Check for existence of user/page
			page, err := repo.GetPage(userName, pageName)
			if err != nil {
				return err
			}

			page.LastModificationDate = i.LastModificationDate

			//Store
			return repo.StorePage(page)
		})
		if err != nil {
			return Item{}, err
		}
	}
	return i, nil
}

//PutItem stores a fully defined item. Replace the item if it already exists
//Returns the stored item.
func (app App) PutItem(userName, pageName string, i Item) (Item, error) {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return Item{}, NotAuthorizedError{"Store item"}
	}

	if len(i.Content) == 0 && len(i.URL) == 0 {
		return Item{}, errors.New("Empty item not allowed. Either content or URL must be provided.")
	}

	//Compute HTML from markdown
	i.HTMLContent = template.HTML(markdownToHTML(i.Content))

	//Stores the item
	err := app.repository.RunInTransaction(func(repo Repository) error {
		//Store
		return repo.StoreItem(userName, pageName, i)
	})
	if err != nil {
		return Item{}, err
	}

	{
		//Update LastModificationDate on Page
		err := app.repository.RunInTransaction(func(repo Repository) error {
			//Check for existence of user/page
			page, err := repo.GetPage(userName, pageName)
			if err != nil {
				return err
			}

			page.LastModificationDate = i.LastModificationDate

			//Store
			return repo.StorePage(page)
		})
		if err != nil {
			return Item{}, err
		}
	}
	return i, nil
}

//UpdateItem stores an updated item
func (app App) UpdateItem(userName, pageName string, i Item, updateTags bool) (Item, error) {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return Item{}, NotAuthorizedError{"Store item"}
	}

	if len(i.Content) == 0 && len(i.URL) == 0 {
		return Item{}, errors.New("Empty item not allowed. Either content or URL must be provided.")
	}

	//Compute HTML from markdown
	i.HTMLContent = template.HTML(markdownToHTML(i.Content))

	i.LastModificationDate = time.Now()

	//Stores the item
	err := app.repository.RunInTransaction(func(repo Repository) error {

		oldItem, err := repo.GetItem(userName, pageName, i.ID)
		if err != nil {
			return err
		}

		i.CreationDate = oldItem.CreationDate

		if !updateTags {
			i.Tags = oldItem.Tags
		}

		//Store
		return repo.StoreItem(userName, pageName, i)
	})
	if err != nil {
		return Item{}, err
	}

	{
		//Update LastModificationDate on Page
		err := app.repository.RunInTransaction(func(repo Repository) error {
			//Check for existence of user/page
			page, err := repo.GetPage(userName, pageName)
			if err != nil {
				return err
			}

			page.LastModificationDate = i.LastModificationDate

			//Store
			return repo.StorePage(page)
		})
		if err != nil {
			return Item{}, err
		}
	}
	return i, nil
}

//SetItemTag stores a new value for an item tag
func (app App) SetItemTag(userName, pageName string, itemID string, tagKey string, tagValue string) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return NotAuthorizedError{"Store item"}
	}

	if len(tagKey) == 0 {
		return errors.New("Empty tag key not allowed.")
	}

	tNow := time.Now()

	//Stores the item
	err := app.repository.RunInTransaction(func(repo Repository) error {

		oldItem, err := repo.GetItem(userName, pageName, itemID)
		if err != nil {
			return err
		}

		oldItem.LastModificationDate = tNow

		oldItem.Tags.SetTag(tagKey, tagValue)

		//Store
		return repo.StoreItem(userName, pageName, oldItem)
	})
	if err != nil {
		return err
	}

	{
		//Update LastModificationDate on Page
		err := app.repository.RunInTransaction(func(repo Repository) error {
			//Check for existence of user/page
			page, err := repo.GetPage(userName, pageName)
			if err != nil {
				return err
			}

			page.LastModificationDate = tNow

			//Store
			return repo.StorePage(page)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

//DeleteItem removes permanently an item
func (app App) DeleteItem(userName, pageName string, itemID string) error {
	currentUserName := app.CurrentUserName()
	//We can only update owned pages
	if len(currentUserName) == 0 || currentUserName != userName {
		return NotAuthorizedError{"Delete page"}
	}

	//Delete item
	err := app.repository.DeleteItem(userName, pageName, itemID)
	if err != nil {
		return err
	}

	return nil
}

//listItems get the list of items for a given page. No authorisation check (should be done before by the caller)
func (app App) listItems(userName, pageName string, limit int) ([]Item, error) {

	//Get the items
	items, err := app.repository.GetItemsFromPage(userName, pageName, limit)
	if err != nil {
		return nil, err
	}

	return items, nil
}

//getItem retrieve a specific item. No authorisation check (should be done before by the caller)
func (app App) getItem(userName, pageName string, itemID string) (Item, error) {
	return app.repository.GetItem(userName, pageName, itemID)
}

//StoreTemplate insert or update a template in database
func (app App) StoreTemplate(tpl Template) error {

	if !app.userInteractor.CurrentUserIsAdmin() {
		return NotAuthorizedError{"Store template"}
	}

	_, err := app.repository.StoreTemplate(tpl, generateID)

	return err
}

//GetTemplate retrieve a template from the datastore
func (app App) GetTemplate(templateID string) (Template, error) {
	return app.repository.GetTemplate(templateID)
}

//GetAllTemplates retrieve all templates from the datastore
func (app App) GetAllTemplates() ([]Template, error) {
	return app.repository.GetAllTemplates()
}

//UploadURL returns the URL to the upload ressource
func (app App) UploadURL(destURL string) (string, error) {
	return app.uploadInteractor.UploadURL(destURL, 10000000) //TODO quotas per user
}

//StoreImage stores an uplaoded image in the datastore and associate it with the current user
func (app App) StoreImage(r *http.Request, name string) error {
	identity, err := app.CurrentIdentity()
	if err != nil {
		return err
	}

	img, err := app.uploadInteractor.UploadInfo(r, name)
	if err != nil {
		return err
	}

	return app.repository.StoreImage(img, identity.UserName)
}

//RenameImage changes the name of an uploaded image
func (app App) RenameImage(imgID string, newName string) error {
	identity, err := app.CurrentIdentity()
	if err != nil {
		return err
	}

	return app.repository.RenameImage(newName, imgID, identity.UserName)
}

//DeleteImage delete an uploaded image
func (app App) DeleteImage(imgID string) error {
	identity, err := app.CurrentIdentity()
	if err != nil {
		return err
	}

	return app.repository.DeleteImage(imgID, identity.UserName)
}

//Images retrieves the images associated with the current user
func (app App) Images(limit int) ([]UploadInfo, error) {
	identity, err := app.CurrentIdentity()
	if err != nil {
		return nil, err
	}

	return app.repository.GetImages(identity.UserName, limit)
}

//ImageURL retrieves the image URL associated with the image
func (app App) ImageURL(img UploadInfo, secure bool, size int) (string, error) {
	return app.uploadInteractor.ImageURL(img.Key, secure, size)
}

//Version returns a user readable text describing the current version of the application
func (app App) Version() string {
	return "1.0-alpha7"
}
