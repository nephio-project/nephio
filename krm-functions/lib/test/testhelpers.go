/*
Copyright 2023 The Nephio Authors.

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
package test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
)

func MustParseKubeObjects(t *testing.T, path string) fn.KubeObjects {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	b := testhelpers.MustReadFile(t, path)

	objects, err := fn.ParseKubeObjects(b)
	if err != nil {
		t.Fatalf("failed to parse objects from file %q: %v", path, err)
	}
	return objects
}

func MustParseKubeObject(t *testing.T, path string) *fn.KubeObject {
	b := testhelpers.MustReadFile(t, path)
	object, err := fn.ParseKubeObject(b)
	if err != nil {
		t.Fatalf("failed to parse object from file %q: %v", path, err)
	}
	return object
}

func InsertBeforeExtension(origPath string, toInsert string) string {
	ext := filepath.Ext(origPath)
	base, _ := strings.CutSuffix(origPath, ext)
	return base + toInsert + ext

}
