// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"math/rand"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/xeonx/timeago"
)

func init() {
	rand.Seed(time.Now().Unix())
}

const (
	idSize    = 18
	charsInID = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func generateID() string {
	b := make([]byte, idSize)
	for i := 0; i < idSize; i++ {
		b[i] = charsInID[rand.Intn(len(charsInID))]
	}
	return string(b)
}

func markdownToHTML(input string) string {

	// set up the HTML renderer
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	// set up the parser
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_HARD_LINE_BREAK

	markdown := blackfriday.Markdown([]byte(input), renderer, extensions)
	html := bluemonday.UGCPolicy().SanitizeBytes(markdown)
	return string(html)
}

func convertToTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}
