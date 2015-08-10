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

func TestPrepareDriverMissingMaster(t *testing.T) {
	leader := ""
	master = &leader

	_, err := prepareDriver(nil, nil, nil, nil)
	assert.NotNil(t, err)
}

func TestPrepareDriverStaticMaster(t *testing.T) {
	leader := "1.2.3.4:5050"
	master = &leader
	m := &MockLeaderDetector{}

	d, err := prepareDriver(nil, m, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, d.Master, "1.2.3.4:5050")
	m.AssertNotCalled(t, "Detector")
}

func TestPrepareDriverZkMaster(t *testing.T) {
	zkUrl := "zk://foo:2181/mesos"
	master = &zkUrl
	leader := "1.2.3.4:5050"

	m := &MockLeaderDetector{}
	m.On("Detect", &zkUrl).Return(&leader, nil)

	d, err := prepareDriver(nil, m, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, d.Master, "1.2.3.4:5050")
	m.AssertExpectations(t)
}
