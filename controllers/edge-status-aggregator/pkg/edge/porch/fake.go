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

package porch

import (
	"context"
	"fmt"
)

type params struct {
	contents   map[string]string
	err        error
	retryCount int
}

const identifierFormat = "%v-%v"

type FakeClient struct {
	// box contains params mapped to a "packageName-clusterName" string
	box chan map[string]*params
}

func NewFakeClient() *FakeClient {
	box := make(chan map[string]*params, 1)
	box <- make(map[string]*params)
	return &FakeClient{
		box: box,
	}
}

func (f *FakeClient) GetRetryCount(ctx context.Context, packageName, clusterName string) (int, error) {
	var p map[string]*params
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case p = <-f.box:
	}

	defer func() {
		f.box <- p
	}()

	var (
		packageParams *params
		ok            bool
	)
	id := fmt.Sprintf(identifierFormat, packageName, clusterName)
	if packageParams, ok = p[id]; !ok {
		return 0, nil
	}
	return packageParams.retryCount, nil
}

func (f *FakeClient) GetContent(ctx context.Context, packageName, clusterName string) (
	map[string]string, error) {
	var p map[string]*params
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p = <-f.box:
	}

	defer func() {
		f.box <- p
	}()

	var (
		packageParams *params
		ok            bool
	)
	id := fmt.Sprintf(identifierFormat, packageName, clusterName)
	if packageParams, ok = p[id]; !ok {
		return nil, fmt.Errorf("package not found")
	}

	deepCopy := make(map[string]string, len(packageParams.contents))
	for k, v := range packageParams.contents {
		deepCopy[k] = v
	}

	return deepCopy, nil
}

func (f *FakeClient) SetError(ctx context.Context, packageName, clusterName string,
	err error) error {
	var p map[string]*params
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p = <-f.box:
	}

	defer func() {
		f.box <- p
	}()

	id := fmt.Sprintf(identifierFormat, packageName, clusterName)
	if _, ok := p[id]; !ok {
		p[id] = &params{}
	}
	p[id].err = err
	return nil
}

func (f *FakeClient) ApplyPackage(ctx context.Context, contents map[string]string, packageName, clusterName string) error {
	var p map[string]*params
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p = <-f.box:
	}

	defer func() {
		f.box <- p
	}()

	id := fmt.Sprintf(identifierFormat, packageName, clusterName)
	if _, ok := p[id]; !ok {
		p[id] = &params{}
	}

	if p[id].err != nil {
		p[id].retryCount++
		return p[id].err
	}

	p[id].retryCount = 0
	p[id].contents = contents

	return nil
}

var _ Client = &FakeClient{}
