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
