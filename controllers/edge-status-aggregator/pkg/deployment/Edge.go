/*
Copyright 2022-2023 The Nephio Authors.

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

package deployment

type Edge struct {
	FirstNode  string
	SecondNode string
}

func (e *Edge) IsEqual(firstNode string, secondNode string) bool {
	if firstNode == e.FirstNode && secondNode == e.SecondNode {
		return true
	} else if secondNode == e.FirstNode && firstNode == e.SecondNode {
		return true
	} else {
		return false
	}
}
