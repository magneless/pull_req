package core

import "errors"

var ErrAlreadyExists = errors.New("resource already exists")
var ErrNotFound = errors.New("resource is not found")
var ErrAlredyMerged = errors.New("pr merged")
var ErrNotAssigned = errors.New("user not a reviewer")
var ErrNoCandidate = errors.New("no candidates")
