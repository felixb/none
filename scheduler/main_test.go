/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareContainerInfoNil(t *testing.T) {
	dockerImage = nil
	containerJson = nil

	assert.Nil(t, prepareContainer(), "ContainerInfo missing")
}

func TestPrepareContainerInfoWithDockerImage(t *testing.T) {
	img := "foo"
	dockerImage = &img
	containerJson = nil

	ci := prepareContainer()
	assert.NotNil(t, ci, "ContainerInfo missing")
	assert.Equal(t, "foo", ci.GetDocker().GetImage(), "Unexpected image")
}

func TestPrepareContainerInfoWithContainerJson(t *testing.T) {
	dockerImage = nil

	b, err := ioutil.ReadFile("../fixtures/container.json")
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, b, "Content missing")
	s := string(b)
	containerJson = &s

	ci := prepareContainer()
	assert.NotNil(t, ci, "ContainerInfo missing")
	assert.Equal(t, "group/image", ci.GetDocker().GetImage(), "Unexpected image")
}

func TestPrepareContainerInfoWithContainerJsonAndDockerImage(t *testing.T) {
	img := "foo"
	dockerImage = &img

	b, err := ioutil.ReadFile("../fixtures/container.json")
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, b, "Content missing")
	s := string(b)
	containerJson = &s

	ci := prepareContainer()
	assert.NotNil(t, ci, "ContainerInfo missing")
	assert.Equal(t, "group/image", ci.GetDocker().GetImage(), "Unexpected image")
}
