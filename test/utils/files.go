/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var filesource FileSource

// SetFileSource registers a provider for files that may be needed at
// runtime. Should be called during initialization of a test binary.
func SetFileSource(source FileSource) {
	filesource = source
}

// FileSource implements one way of retrieving test file content.  For
// example, one file source could read from the original source code
// file tree, another from bindata compiled into a test executable.
type FileSource interface {
	// When the file is not found, a nil slice is returned.
	// An error is returned for all fatal errors.
	ReadTestFile(filePath string) ([]byte, error)

	// DescribeFiles returns a multi-line description of which
	// files are available via this source. It is meant to be
	// used as part of the error message when a file cannot be
	// found.
	DescribeFiles() string

	// GetAbsPath returns the full path of a file
	// An error is returned for all fatal errors.
	GetAbsPath(filePath string) (string, error)
}

// Read tries to retrieve the desired file content from
// one of the registered file sources.
func Read(filePath string) ([]byte, error) {
	if filesource == nil {
		return nil, fmt.Errorf("no file sources registered (yet?), cannot retrieve test file %s", filePath)
	}

	data, err := filesource.ReadTestFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("fatal error retrieving test file %s: %s", filePath, err)
	}

	if data != nil {
		return data, nil
	}

	// Here we try to generate an error that points test authors
	// or users in the right direction for resolving the problem.
	error := fmt.Sprintf("Test file %q was not found.\n", filePath)
	error += filesource.DescribeFiles()
	error += "\n"

	return nil, errors.New(error)
}

// Exists checks whether a file could be read. Unexpected errors
// are handled by calling the fail function, which then should
// abort the current test.
func Exists(filePath string) bool {
	data, err := filesource.ReadTestFile(filePath)
	if err != nil {
		// log error?
		return false
	}

	if data != nil {
		return true
	}

	return false
}

// RootFileSource looks for files relative to a root directory.
type RootFileSource struct {
	Root string
}

// ReadTestFile looks for the file relative to the configured
// root directory. If the path is already absolute, for example
// in a test that has its own method of determining where
// files are, then the path will be used directly.
func (r RootFileSource) ReadTestFile(filePath string) ([]byte, error) {
	var fullPath string

	if path.IsAbs(filePath) {
		fullPath = filePath
	} else {
		fullPath = filepath.Join(r.Root, filePath)
	}

	data, err := ioutil.ReadFile(fullPath)
	if os.IsNotExist(err) {
		return nil, err
	}

	return data, err
}

// DescribeFiles explains that it looks for files inside
// a certain root directory.
func (r RootFileSource) DescribeFiles() string {
	description := fmt.Sprintf("Test files are expected in %q", r.Root)

	if !path.IsAbs(r.Root) {
		abs, err := filepath.Abs(r.Root)
		if err == nil {
			description += fmt.Sprintf(" = %q", abs)
		}
	}

	description += "."

	return description
}

func (r RootFileSource) GetAbsPath(filePath string) (string, error) {
	var fullPath string

	if path.IsAbs(filePath) {
		fullPath = filePath
	} else {
		fullPath = filepath.Join(r.Root, filePath)
	}

	_, err := os.Stat(fullPath)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}
