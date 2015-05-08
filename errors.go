// Copyright 2014-2015 Simon HEGE. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package okinotes

import (
	"fmt"
)

//DataError represents a validation error on data
type DataError struct {
	Field   string
	Message string
}

func (err DataError) Error() string {
	return fmt.Sprintf("Unvalid %s: %s", err.Field, err.Message)
}

//NotAuthorizedError represents an authorisation error on operation
type NotAuthorizedError struct {
	Operation string
}

func (err NotAuthorizedError) Error() string {
	return fmt.Sprintf("%s not permitted.", err.Operation)
}
