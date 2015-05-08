// Copyright 2014 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ae

import (
	"errors"
	"net/http"

	"appengine"
	"appengine/blobstore"
	"appengine/image"

	"github.com/okinotes/okinotes"
)

type uploadInteractor struct {
	c appengine.Context
}

func (i uploadInteractor) UploadURL(destURL string, maxUploadBytes int64) (string, error) {
	url, err := blobstore.UploadURL(i.c, destURL, &blobstore.UploadURLOptions{MaxUploadBytes: maxUploadBytes})
	if err != nil {
		return "", err
	}

	return url.String(), err
}
func (i uploadInteractor) UploadInfo(r *http.Request, name string) (okinotes.UploadInfo, error) {
	blobs, _, err := blobstore.ParseUpload(r)
	if err != nil {
		return okinotes.UploadInfo{}, err
	}
	file := blobs[name]
	if len(file) == 0 {
		return okinotes.UploadInfo{}, errors.New("No file uploaded")
	}

	return okinotes.UploadInfo{
		Key:          string(file[0].BlobKey),
		ContentType:  file[0].ContentType,
		CreationTime: file[0].CreationTime,
		Filename:     file[0].Filename,
		Size:         file[0].Size,
	}, nil
}
func (i uploadInteractor) ImageURL(key string, secure bool, size int) (string, error) {
	url, err := image.ServingURL(i.c, appengine.BlobKey(key), &image.ServingURLOptions{Secure: secure, Size: size})
	if err != nil {
		return "", err
	}

	return url.String(), err
}
func (i uploadInteractor) Delete(key string) error {
	_ = image.DeleteServingURL(i.c, appengine.BlobKey(key))

	return blobstore.Delete(i.c, appengine.BlobKey(key))
}
