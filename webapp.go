// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/tools/blog/atom"
)

var allTemplates *template.Template

func init() {
	funcMap := template.FuncMap{
		//"title": strings.Title,
		//"cssColor": cssColor,
		"timeago": convertToTimeAgo,
	}
	allTemplates = template.Must(template.New("allTemplates").Funcs(funcMap).ParseGlob("resources/templates/*.tpl"))
}

//RegisterPagesOnRouter initializes the router for the pages of the webapp
func RegisterPagesOnRouter(m *mux.Router, f AppFactory) error {

	handlers := map[string]map[string]http.HandlerFunc{
		"GET": {
			//Static pages
			"/":                      makePageHandler(getIndexHTML, f),
			"/index.html":            makePageHandler(getIndexHTML, f),
			"/first_connection.html": makePageHandler(getFirstConnectionHTML, f),
			"/help.html":             makeStaticPageHandler("help", f),
			"/about.html":            makeStaticPageHandler("about", f),
			"/roadmap.html":          makeStaticPageHandler("roadmap", f),
			//User images
			"/user/images.html": makePageHandler(pageImages, f),
			//Page administration
			"/administrate.html":    makePageHandler(pageAdminGet, f),
			"/change_template.html": makePageHandler(pageChangeTemplateGet, f),
			"/delete.html":          makePageHandler(pageDeleteGet, f),
			"/newItem.html":         makePageHandler(pageNewItemGet, f),
			"/editItem.html":        makePageHandler(pageEditItemGet, f),
			"/deleteItem.html":      makePageHandler(pageDeleteItemGet, f),
			"/importPage.html":      makePageHandler(pageImportPageGet, f),
			//Pages
			"/p/{userName}/{pageName}.html":                       makePageHandler(pagePage, f),
			"/p/{userName}/{pageName}/offline.html":               makePageHandler(offlinePage, f),
			"/p/{userName}/{pageName}/cache.manifest":             makePageHandler(cacheManifestPage, f),
			"/p/{userName}/{pageName}/atom.xml":                   makePageHandler(xmlPage, f),
			"/p/{userName}/{pageName}/{userName}_{pageName}.json": makePageHandler(jsonPage, f),
			//Images
			"/images/{imgID}": makePageHandler(getImage, f),
			//Administration
			"/templates.htm": makePageHandler(pageAdminTemplates, f),
		},
		"POST": {
			//Static pages
			"/first_connection.html": makePageHandler(postFirstConnectionHTML, f),
			//User images
			"/user/images.html":        makePageHandler(pageImagesPost, f),
			"/user/images/rename.html": makePageHandler(pageImageRename, f),
			"/user/images/delete.html": makePageHandler(pageImageDelete, f),
			//Page administration
			"/create.html":          makePageHandler(pageCreatePost, f),
			"/administrate.html":    makePageHandler(pageAdminPost, f),
			"/change_template.html": makePageHandler(pageChangeTemplatePost, f),
			"/delete.html":          makePageHandler(pageDeletePost, f),
			"/deleteItem.html":      makePageHandler(pageDeleteItemPost, f),
			"/importPage.html":      makePageHandler(pageImportPagePost, f),
			//Pages
			"/p/{userName}/{pageName}.html": makePageHandler(pageAddItem, f),
		},
		"DELETE": {},
		"OPTION": {},
	}

	for method, routes := range handlers {
		for route, fct := range routes {
			m.HandleFunc(route, fct).Methods(method)
		}
	}

	return nil
}

//RegisterAPIOnRouter initializes the router for the API of the webapp
func RegisterAPIOnRouter(m *mux.Router, f AppFactory) error {

	m.HandleFunc("/version", makeAppHandler(getVersion, f, http.StatusOK)).Methods("GET")
	m.HandleFunc("/currentUser", makeAppHandler(getCurrentUser, f, http.StatusOK)).Methods("GET")

	m.HandleFunc("/users/{userName}/pages/{pageName}", makeAppHandler(getPage, f, http.StatusOK)).Methods("GET")

	m.HandleFunc("/users/{userName}/pages/{pageName}/items", makeAppHandler(getItems, f, http.StatusOK)).Methods("GET")
	m.HandleFunc("/users/{userName}/pages/{pageName}/items", makeAppHandler(createItem, f, http.StatusCreated)).Methods("POST")

	m.HandleFunc("/users/{userName}/pages/{pageName}/items/{itemID}", makeAppHandler(editItem, f, http.StatusAccepted)).Methods("POST")
	m.HandleFunc("/users/{userName}/pages/{pageName}/items/{itemID}", makeAppHandler(putItem, f, http.StatusOK)).Methods("PUT")
	m.HandleFunc("/users/{userName}/pages/{pageName}/items/{itemID}", makeAppHandler(deleteItem, f, http.StatusOK)).Methods("DELETE")

	return nil
}

func handleError(w http.ResponseWriter, r *http.Request, err error, app App) {
	logger := app.logInteractor
	logger.Errorf("%v", err)

	if _, ok := err.(NotAuthorizedError); ok && app.userInteractor != nil {
		if ident, execErr := app.userInteractor.CurrentIdentity(); execErr == nil && len(ident.Identity) == 0 {
			loginURL, execErr := app.userInteractor.LoginURL(r.URL.String())
			if execErr != nil {
				logger.Errorf("%v", execErr)
				http.Error(w, execErr.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}
	}
	var execErr error

	data := struct {
		sharedData
		ErrorCode    int
		ErrorMessage string
	}{}

	execErr = data.init("home", "index.html", "index.html", app)
	if execErr != nil {
		logger.Errorf("%v", execErr)
		http.Error(w, execErr.Error(), http.StatusInternalServerError)
	}

	if _, ok := err.(NotInDatastoreError); ok {
		data.ErrorCode = http.StatusNotFound
	} else if _, ok := err.(NotAuthorizedError); ok {
		data.ErrorCode = http.StatusUnauthorized
	} else {
		data.ErrorCode = http.StatusInternalServerError
	}
	data.ErrorMessage = err.Error()

	execErr = allTemplates.ExecuteTemplate(w, "error.html.tpl", &data)
	if execErr != nil {
		logger.Errorf("%v", execErr)
		http.Error(w, execErr.Error(), http.StatusInternalServerError)
	}
}

func makeAppHandler(fn func(*http.Request, App) (interface{}, error), f AppFactory, statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := f.CreateApp(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := fn(r, app)
		if err != nil {
			handleError(w, r, err, app)
			return
		}

		if data == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(statusCode)
			encoder := json.NewEncoder(w)
			err := encoder.Encode(data)
			if err != nil {
				handleError(w, r, err, app)
				return
			}
		}
	}
}

func makeStaticPageHandler(page string, f AppFactory) http.HandlerFunc {

	pageURL := fmt.Sprintf("/%s.html", page)
	pageTemplateFile := fmt.Sprintf("%s.html.tpl", page)

	return makePageHandler(func(r *http.Request, app App) (handler, error) {

		var err error

		data := sharedData{}

		err = data.init(page, pageURL, pageURL, app)
		if err != nil {
			return nil, err
		}

		return templateHandler{pageTemplateFile, data}, nil

	}, f)
}
func makePageHandler(fn func(*http.Request, App) (handler, error), f AppFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		app, err := f.CreateApp(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		c, err := fn(r, app)
		if err != nil {
			handleError(w, r, err, app)
			return
		}
		if c == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			err = c.ServeHTTP(w, r)
			if err != nil {
				handleError(w, r, err, app)
				return
			}
		}
	}
}

type sharedData struct {
	Nav       string
	User      User
	Version   string
	LoginURL  string
	LogoutURL string

	Templates []Template
}

func (data *sharedData) init(nav string, loginRedirectURL string, logoutRedirectURL string, app App) error {
	var err error

	data.Nav = nav

	data.User, err = app.CurrentUser()
	if err != nil && err != ErrFirstUserConnection {
		app.logInteractor.Debugf("init: err = %v", err)
		return err
	}
	data.Version = app.Version()
	data.LoginURL, err = app.LoginURL(loginRedirectURL)
	if err != nil {
		return err
	}
	data.LogoutURL, err = app.LogoutURL(logoutRedirectURL)
	if err != nil {
		return err
	}
	data.Templates, err = app.GetAllTemplates()
	if err != nil {
		return err
	}
	return nil
}

func getIndexHTML(r *http.Request, app App) (handler, error) {
	var err error

	data := struct {
		sharedData
		MyPages     []Page
		MoreMyPages bool
	}{}

	err = data.init("home", "index.html", "index.html", app)
	if err != nil {
		return nil, err
	}

	identity, err := app.CurrentIdentity()
	if err == ErrFirstUserConnection {
		return redirectHandler{"/first_connection.html"}, nil
	}
	if err != nil {
		return nil, err
	}

	if len(identity.Identity) > 0 {
		data.MyPages, data.MoreMyPages, err = app.ListOwnedPages(10) //TODO: limit?
		if err != nil {
			return nil, err
		}
	}

	return templateHandler{"index.html.tpl", data}, nil
}

func getFirstConnectionHTML(r *http.Request, app App) (handler, error) {
	//Redirect to index if not logged in
	ident, err := app.CurrentIdentity()
	if err != ErrFirstUserConnection {
		if err == nil {
			return redirectHandler{"/index.html"}, nil
		}
		return nil, err
	}

	if len(ident.Identity) == 0 {
		return redirectHandler{"/index.html"}, nil
	}

	data := struct {
		sharedData

		Identity     string
		ErrorMessage string
	}{}

	err = data.init("home", "/index.html", "/index.html", app)
	if err != nil {
		return nil, err
	}

	data.Identity = ident.Identity

	return templateHandler{"first_connection.html.tpl", data}, nil
}
func postFirstConnectionHTML(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("name")

	//Redirect to index if not logged in
	identity, err := app.CurrentIdentity()
	if err != ErrFirstUserConnection {
		if err == nil {
			return redirectHandler{"/index.html"}, nil
		}
		return nil, err
	}

	if len(identity.Identity) == 0 {
		return redirectHandler{"/index.html"}, nil
	}

	err = app.CreateUser(identity.Ident, userName)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/index.html"}, nil
}

type img struct {
	ID           string
	Name         string
	URL128       string
	URL          string
	CreationTime time.Time
}

func pageImages(r *http.Request, app App) (handler, error) {
	var err error

	data := struct {
		sharedData
		UploadURL string
		Images    []img
	}{}

	err = data.init("media.images", "/user/images.html", "/user/images.html", app)
	if err != nil {
		return nil, err
	}

	data.UploadURL, err = app.UploadURL("/user/images.html")
	if err != nil {
		return nil, err
	}

	imgs, err := app.Images(1000)
	if err != nil {
		return nil, err
	}
	for _, i := range imgs {
		url128, err := app.ImageURL(i, false, 128)
		if err != nil {
			return nil, err
		}
		url, err := app.ImageURL(i, false, 0)
		if err != nil {
			return nil, err
		}

		data.Images = append(data.Images, img{
			ID:           i.Key,
			Name:         i.Filename,
			URL128:       url128,
			URL:          url,
			CreationTime: i.CreationTime,
		})
	}

	return templateHandler{"images.html.tpl", data}, nil
}
func pageImagesPost(r *http.Request, app App) (handler, error) {

	err := app.StoreImage(r, "file")
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/user/images.html"}, nil
}
func pageImageRename(r *http.Request, app App) (handler, error) {
	imgID := r.FormValue("imgID")
	newName := r.FormValue("imageName")

	err := app.RenameImage(imgID, newName)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/user/images.html"}, nil
}
func pageImageDelete(r *http.Request, app App) (handler, error) {
	imgID := r.FormValue("imgID")

	err := app.DeleteImage(imgID)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/user/images.html"}, nil
}

func getImage(r *http.Request, app App) (handler, error) {
	vars := mux.Vars(r)
	imgID := vars["imgID"]

	if imgID == "default" {
		return redirectHandler{"/static/img/bg-pehoe.jpg"}, nil
	}

	url, err := app.uploadInteractor.ImageURL(imgID, false, 0)
	if err != nil {
		return nil, err
	}

	return redirectHandler{url}, nil
}

func pageCreatePost(r *http.Request, app App) (handler, error) {
	pageName := r.FormValue("pageName")
	templateID := r.FormValue("template")
	user, err := app.CurrentUser()
	if err != nil {
		return nil, err
	}

	if len(user.Name) == 0 {
		loginURL, err := app.LoginURL("/create.html?pageName=" + pageName + "&template=" + templateID)
		if err != nil {
			return nil, err
		}

		return redirectHandler{loginURL}, nil
	}

	if _, err := app.GetTemplate(templateID); err != nil {
		templateID = "blog2col"
	}

	page := Page{
		UserName:   user.Name,
		Name:       pageName,
		Title:      pageName,
		TemplateID: templateID,
	}

	err = app.CreatePage(page)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/p/" + user.Name + "/" + pageName + ".html#administrate"}, nil
}

func pageAdminGet(r *http.Request, app App) (handler, error) {

	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}
	template, err := app.GetTemplate(page.TemplateID)
	if err != nil {
		return nil, err
	}
	page.Tags.DefaultTo(template.PageTags)

	data := struct {
		sharedData
		Page     Page
		Template Template
	}{}
	err = data.init("", "/p/"+userName+"/"+pageName+".html", "index.html", app)
	if err != nil {
		return nil, err
	}
	data.Page = page
	data.Template = template

	return templateHandler{"dlg_administrate.html.tpl", data}, nil
}
func pageAdminPost(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	newTitle := r.FormValue("title")
	newContentLicense := r.FormValue("contentLicense")
	newPolicy := Policy(r.FormValue("policy"))

	//Get tags
	page, err := app.GetPage(userName, pageName)
	if err != nil {
		errNotFound, notFound := err.(NotInDatastoreError)
		if notFound && errNotFound.Type == "Page" {
			return redirectHandler{"/create.html?userName=" + userName + "&pageName=" + pageName}, nil
		}
		return nil, err
	}
	template, err := app.GetTemplate(page.TemplateID)
	if err != nil {
		return nil, err
	}

	newPage := Page{
		UserName:       userName,
		Name:           pageName,
		Title:          newTitle,
		ContentLicense: newContentLicense,
		Policy:         newPolicy,
		TemplateID:     page.TemplateID, //TODO: editable ?
	}

	for _, tag := range template.PageTags {
		value := r.FormValue(fmt.Sprintf("tag-%s", tag.Key))
		if len(value) > 0 && value != tag.DefaultValue {
			newPage.Tags = append(newPage.Tags, Tag{tag.Key, value})
		}
	}

	err = app.UpdatePage(newPage, template.PageTags)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/p/" + userName + "/" + pageName + ".html"}, nil
}

func pageChangeTemplateGet(r *http.Request, app App) (handler, error) {

	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}
	templates, err := app.GetAllTemplates()
	if err != nil {
		return nil, err
	}

	data := struct {
		sharedData
		UserName     string
		PageName     string
		AllTemplates []struct {
			Template
			Selected bool
		}
		Selected int
	}{}
	err = data.init("", "/p/"+userName+"/"+pageName+".html", "index.html", app)
	if err != nil {
		return nil, err
	}
	data.UserName = userName
	data.PageName = pageName

	for _, t := range templates {
		data.AllTemplates = append(data.AllTemplates, struct {
			Template
			Selected bool
		}{
			t,
			t.ID == page.TemplateID,
		})
	}

	return templateHandler{"dlg_change_template.html.tpl", data}, nil
}
func pageChangeTemplatePost(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	newTemplateID := r.FormValue("templateID")

	err := app.UpdateTemplate(userName, pageName, newTemplateID)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/p/" + userName + "/" + pageName + ".html"}, nil
}

type pageData struct {
	sharedData
	Page     Page
	Template Template
	Offline  bool
	CanEdit  bool
	Items    []Item
}

func pagePage(r *http.Request, app App) (handler, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	page, err := app.GetPage(userName, pageName)
	if err != nil {

		currentUserName := app.CurrentUserName()

		errNotFound, notFound := err.(NotInDatastoreError)
		if notFound && errNotFound.Type == "Page" && currentUserName == userName {
			return redirectHandler{"/create.html?pageName=" + pageName}, nil
		}
		return nil, err
	}

	items, err := app.listItems(page.UserName, page.Name, 1000)
	if err != nil {
		return nil, err
	}

	if app.CurrentUserIsAdmin() {
		debug := r.FormValue("debug")
		if debug == "true" {
			page.Tags.SetTag("template.debug", "true")
		}
	}

	template, err := app.GetTemplate(page.TemplateID)
	if err != nil {
		return nil, err
	}
	page.Tags.DefaultTo(template.PageTags)
	for i := range items {
		items[i].Tags.DefaultTo(template.ItemTags)
	}

	data := pageData{}

	logoutDest := "index.html"
	if page.Policy == PolicyPUBLIC {
		logoutDest = "/p/" + userName + "/" + pageName + ".html"
	}

	err = data.init("", "/p/"+userName+"/"+pageName+".html", logoutDest, app)
	if err != nil {
		return nil, err
	}
	data.Page = page
	data.Template = template
	data.CanEdit = (app.CurrentUserName() == userName)
	data.Items = items

	return templateHandler{template.File, data}, nil
}

func offlinePage(r *http.Request, app App) (handler, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}

	items, err := app.listItems(page.UserName, page.Name, 1000)
	if err != nil {
		return nil, err
	}

	if app.CurrentUserIsAdmin() {
		debug := r.FormValue("debug")
		if debug == "true" {
			page.Tags.SetTag("template.debug", "true")
		}
	}

	template, err := app.GetTemplate(page.TemplateID)
	if err != nil {
		return nil, err
	}
	page.Tags.DefaultTo(template.PageTags)
	for i := range items {
		items[i].Tags.DefaultTo(template.ItemTags)
	}

	data := pageData{}

	logoutDest := "index.html"
	if page.Policy == PolicyPUBLIC {
		logoutDest = "/p/" + userName + "/" + pageName + ".html"
	}

	err = data.init("", "/p/"+userName+"/"+pageName+".html", logoutDest, app)
	if err != nil {
		return nil, err
	}
	data.Offline = true
	data.Page = page
	data.Template = template
	data.CanEdit = (app.CurrentUserName() == userName)
	data.Items = items

	return templateHandler{template.File, data}, nil
}

func cacheManifestPage(r *http.Request, app App) (handler, error) {
	c, err := pagePage(r, app)
	if err != nil {
		return nil, err
	}
	tContent, ok := c.(templateHandler)
	if !ok {
		return nil, errors.New("Conversion failed")
	}
	data, ok := tContent.Data.(pageData)
	if !ok {
		return nil, errors.New("Conversion failed")
	}

	cacheTemplate := "p_cache.manifest.tpl"
	return templateHandler{cacheTemplate, data}, nil
}

func xmlPage(r *http.Request, app App) (handler, error) {
	c, err := pagePage(r, app)
	if err != nil {
		return nil, err
	}
	tContent, ok := c.(templateHandler)
	if !ok {
		return nil, errors.New("Conversion failed")
	}
	data, ok := tContent.Data.(pageData)
	if !ok {
		return nil, errors.New("Conversion failed")
	}

	atomXML := atom.Feed{
		Title: data.Page.Title,
		ID:    "okinotes:page:" + data.Page.UserName + "/" + data.Page.Name,
		//Link: ,
		Updated: atom.Time(data.Page.LastModificationDate),
		Author: &atom.Person{
			Name: data.Page.UserName,
		},
	}

	for _, item := range data.Items {
		entry := atom.Entry{
			Title: item.Title,
			ID:    "okinotes:item:" + item.ID,
			//Link
			Published: atom.Time(item.CreationDate),
			Updated:   atom.Time(item.LastModificationDate),
			//Author
			//Summary
			Content: &atom.Text{
				Type: "html",
				Body: string(item.HTMLContent),
			},
		}

		if len(item.URL) > 0 {
			entry.Link = append(entry.Link, atom.Link{
				Rel:  "related",
				Href: item.URL,
			})
		}
		if len(item.Source) > 0 {
			entry.Author = &atom.Person{
				Name: item.Source,
			}
		}

		atomXML.Entry = append(atomXML.Entry, &entry)
	}

	return marshalHandler{xml.Marshal, atomXML, "application/atom+xml", ""}, nil
}

type pageJSONData struct {
	Page  Page
	Items []Item
}

func jsonPage(r *http.Request, app App) (handler, error) {

	c, err := pagePage(r, app)
	if err != nil {
		return nil, err
	}
	tContent, ok := c.(templateHandler)
	if !ok {
		return nil, errors.New("Conversion failed")
	}
	data, ok := tContent.Data.(pageData)
	if !ok {
		return nil, errors.New("Conversion failed")
	}

	d := pageJSONData{data.Page, data.Items}

	return marshalHandler{json.Marshal, d, "application/json", data.User.Name + "_" + data.Page.Name + ".json"}, nil
}

func pageImportPageGet(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	return templateHandler{"dlg_importPage.html.tpl", struct {
		UserName string
		PageName string
	}{userName, pageName}}, nil
}
func pageImportPagePost(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}

	uploadedContent := pageJSONData{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&uploadedContent); err != nil {
		return nil, err
	}

	template, err := app.GetTemplate(uploadedContent.Page.TemplateID)
	if err != nil {
		return nil, err
	}

	//New page is in uploadedContent. Save page
	uploadedContent.Page.UserName = userName
	uploadedContent.Page.Name = pageName

	err = app.UpdatePage(uploadedContent.Page, template.PageTags)
	if err != nil {
		return nil, err
	}

	//Save items
	for _, item := range uploadedContent.Items {
		_, err = app.UpdateItem(userName, pageName, item, true)
		if err != nil {
			if _, err = app.CreateItem(userName, pageName, item); err != nil {
				return nil, err
			}
		}
	}

	return redirectHandler{"/p/" + userName + "/" + pageName + ".html"}, nil
}

type deleteData struct {
	Page Page
}

func pageDeleteGet(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}

	data := deleteData{}
	data.Page = page

	return templateHandler{"dlg_delete.html.tpl", data}, nil
}
func pageDeletePost(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	err := app.DeletePage(userName, pageName)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/index.html"}, nil
}

type newItemData struct {
	Page Page
}

func pageNewItemGet(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}

	data := newItemData{}
	data.Page = page

	return templateHandler{"dlg_newItem.html.tpl", data}, nil
}

type editItemData struct {
	Page Page
	Item Item
}

func pageEditItemGet(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")
	itemID := r.FormValue("itemID")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}
	item, err := app.getItem(userName, pageName, itemID)
	if err != nil {
		return nil, err
	}

	data := editItemData{}
	data.Page = page
	data.Item = item

	return templateHandler{"dlg_editItem.html.tpl", data}, nil
}

func pageAddItem(r *http.Request, app App) (handler, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	item := Item{
		Kind:    r.FormValue("kind"),
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
		Source:  r.FormValue("source"),
		URL:     r.FormValue("URL"),
	}

	_, err := app.CreateItem(userName, pageName, item)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/p/" + userName + "/" + pageName + ".html"}, nil
}

type deleteItemData struct {
	Page Page
	Item Item
}

func pageDeleteItemGet(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")
	itemID := r.FormValue("itemID")

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}

	item, err := app.getItem(userName, pageName, itemID)
	if err != nil {
		return nil, err
	}

	data := deleteItemData{}
	data.Page = page
	data.Item = item

	return templateHandler{"dlg_deleteItem.html.tpl", data}, nil
}
func pageDeleteItemPost(r *http.Request, app App) (handler, error) {
	userName := r.FormValue("userName")
	pageName := r.FormValue("pageName")
	itemID := r.FormValue("itemID")

	err := app.DeleteItem(userName, pageName, itemID)
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/p/" + userName + "/" + pageName + ".html"}, nil
}
func pageAdminTemplates(r *http.Request, app App) (handler, error) {

	tNow := time.Now()

	err := app.StoreTemplate(Template{
		ID: "blog2col",

		Name: "Micro-blog",
		File: "p_blog2col.html.tpl",

		CreationDate:         tNow,
		LastModificationDate: tNow,

		PageTags: TagDescriptionList{
			TagDescription{"background", "Background image", "imageId", "The location of the image to be used as a background for the page.", "default"},
			TagDescription{"title.color", "Title color", "color", "The color to be used for the page title.", "#000000"},
		},

		ItemTags: TagDescriptionList{},
	})
	if err != nil {
		return nil, err
	}

	err = app.StoreTemplate(Template{
		ID: "urllist",

		Name: "URL list",
		File: "p_urllist.html.tpl",

		CreationDate:         tNow,
		LastModificationDate: tNow,

		PageTags: TagDescriptionList{
			TagDescription{"background", "Background image", "imageId", "The location of the image to be used as a background for the page.", "default"},
			TagDescription{"title.color", "Title color", "color", "The color to be used for the page title.", "#000000"},
		},

		ItemTags: TagDescriptionList{},
	})
	if err != nil {
		return nil, err
	}

	err = app.StoreTemplate(Template{
		ID: "angular",

		Name: "Micro-blog with offline mode",
		File: "p_angular.html.tpl",

		CreationDate:         tNow,
		LastModificationDate: tNow,

		PageTags: TagDescriptionList{
			TagDescription{"background", "Background image", "imageId", "The location of the image to be used as a background for the page.", "default"},
			TagDescription{"title.color", "Title color", "color", "The color to be used for the page title.", "#000000"},
		},

		ItemTags: TagDescriptionList{},
	})
	if err != nil {
		return nil, err
	}

	err = app.StoreTemplate(Template{
		ID: "todolist",

		Name: "TODO list",
		File: "p_todolist.html.tpl",

		CreationDate:         tNow,
		LastModificationDate: tNow,

		PageTags: TagDescriptionList{
			TagDescription{"background", "Background image", "imageId", "The location of the image to be used as a background for the page.", "default"},
			TagDescription{"title.color", "Title color", "color", "The color to be used for the page title.", "#000000"},
		},

		ItemTags: TagDescriptionList{
			TagDescription{"status", "Status", "_status", "The status of the item (new, done, archived, ...).", "new"},
			TagDescription{"deadline", "Deadline", "datetime", "The deadline for the item", ""},
		},
	})
	if err != nil {
		return nil, err
	}

	return redirectHandler{"/index.html"}, nil
}

func getVersion(r *http.Request, app App) (interface{}, error) {
	type data struct {
		Version string `json:"version"`
	}
	return data{app.Version()}, nil
}

func getCurrentUser(r *http.Request, app App) (interface{}, error) {
	return app.CurrentUser()
}

func getPage(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	page, err := app.GetPage(userName, pageName)
	if err != nil {
		return nil, err
	}

	return page, nil
}

func getItems(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	items, err := app.listItems(userName, pageName, 1000) //TODO: read limit in query + use paging
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []Item{}
	}

	return items, nil
}
func createItem(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]

	newItem := Item{
		Kind:    r.FormValue("kind"),
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
		Source:  r.FormValue("source"),
		URL:     r.FormValue("URL"),
	}

	//Check for JSON body
	if r.Body != nil {
		if body, err := ioutil.ReadAll(r.Body); err == nil {
			var jsonItem Item
			if err := json.Unmarshal(body, &jsonItem); err == nil {
				newItem = jsonItem
			}
		}
	}

	newItem, err := app.CreateItem(userName, pageName, newItem)
	if err != nil {
		return nil, err
	}

	return newItem, nil
}
func putItem(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]
	itemID := vars["itemID"]

	var newItem Item
	//Check for JSON body
	if r.Body != nil {
		if body, err := ioutil.ReadAll(r.Body); err == nil {
			var jsonItem Item
			if err := json.Unmarshal(body, &jsonItem); err == nil {
				newItem = jsonItem
			}
		}
	}
	newItem.ID = itemID

	newItem, err := app.PutItem(userName, pageName, newItem)
	if err != nil {
		return nil, err
	}

	return newItem, nil
}
func editItem(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]
	itemID := vars["itemID"]

	mode := r.FormValue("mode")

	if mode == "updateTag" {

		for key, array := range r.PostForm {
			if key[:4] == "tag-" && len(array) > 0 {
				tagName := key[4:]
				tagValue := array[0]

				err := app.SetItemTag(userName, pageName, itemID, tagName, tagValue)
				if err != nil {
					return nil, err
				}
			}
		}

	} else {
		editedItem := Item{
			ID:      itemID,
			Kind:    r.FormValue("kind"),
			Title:   r.FormValue("title"),
			Content: r.FormValue("content"),
			Source:  r.FormValue("source"),
			URL:     r.FormValue("URL"),
		}
		//Check for JSON body
		if r.Body != nil {
			if body, err := ioutil.ReadAll(r.Body); err == nil {
				var jsonItem Item
				if err := json.Unmarshal(body, &jsonItem); err == nil {
					editedItem = jsonItem
					editedItem.ID = itemID
				}
			}
		}

		i, err := app.UpdateItem(userName, pageName, editedItem, false)
		if err != nil {
			return nil, err
		}
		return i, nil
	}
	return Item{}, nil
}

func deleteItem(r *http.Request, app App) (interface{}, error) {
	vars := mux.Vars(r)
	userName := vars["userName"]
	pageName := vars["pageName"]
	itemID := vars["itemID"]

	err := app.DeleteItem(userName, pageName, itemID)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
