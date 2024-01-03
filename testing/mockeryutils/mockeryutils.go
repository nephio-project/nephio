/*
Copyright 2024 The Nephio Authors.

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

package mockeryutils

import (
	"github.com/stretchr/testify/mock"
)

func InitMocks(mocked *mock.Mock, mocks []MockHelper) {
	if mocked == nil {
		panic("\"mocked\" may not be nil")
	}

	for counter := range mocks {
		call := mocked.On(mocks[counter].MethodName)
		for _, arg := range mocks[counter].ArgType {
			call.Arguments = append(call.Arguments, mock.AnythingOfType(arg))
		}
		for _, ret := range mocks[counter].RetArgList {
			call.ReturnArguments = append(call.ReturnArguments, ret)
		}
	}
}
