// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"sort"
	"time"
)

//UploadInfo contains all the information of uploaded content
type UploadInfo struct {
	Key          string
	ContentType  string
	CreationTime time.Time
	Filename     string
	Size         int64
}

//Template represents a page schema with optional parameters
type Template struct {
	ID string

	CreationDate         time.Time
	LastModificationDate time.Time

	Name string
	File string

	PageTags TagDescriptionList
	ItemTags TagDescriptionList
}

//TagDescription represents metadata on a Tag
type TagDescription struct {
	Key          string
	Name         string
	Kind         string
	Description  string
	DefaultValue string
}

//TagDescriptionList represenets a list of TagDescription
type TagDescriptionList []TagDescription

//Tag represents additional informations allowing dynamic extension of some structs
type Tag struct {
	Key   string
	Value string
}

//TagList represent all the extension information
type TagList []Tag

func (l TagList) Len() int           { return len(l) }
func (l TagList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l TagList) Less(i, j int) bool { return l[i].Key < l[j].Key }

//Tag returns the value associated to the given key. Returns an empty string when the key does not exists.
func (l TagList) Tag(key string) string {
	i := sort.Search(len(l), func(i int) bool { return l[i].Key >= key })
	if i < len(l) && l[i].Key == key {
		return l[i].Value
	}
	return ""
}

//SetTag add or replace the value associated to the given key.
//TagList remains sorted after this operation.
func (l *TagList) SetTag(key, value string) {
	i := sort.Search(len(*l), func(i int) bool { return (*l)[i].Key >= key })
	if i < len(*l) && (*l)[i].Key == key {
		(*l)[i].Value = value
		return
	}
	*l = append(*l, Tag{key, value})
	sort.Sort(l)
}

//DefaultTo add or replace the values of the current tag list with the one from the new list
//TagList remains sorted after this operation.
func (l *TagList) DefaultTo(defaultValues TagDescriptionList) {
	for _, tag := range defaultValues {
		if len(l.Tag(tag.Key)) == 0 {
			l.SetTag(tag.Key, tag.DefaultValue)
		}
	}
}
