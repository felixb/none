package main

import (
	"testing"

	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLeaderDetector struct {
	mock.Mock
}

func (m *MockLeaderDetector) Detect(url *string) (*string, error) {
	args := m.Called(url)
	return args.Get(0).(*string), args.Error(1)
}

func TestOnLeaderChangeIp(t *testing.T) {
	d := NewZkLeaderDetector()
	mi := util.NewMasterInfo("id", 0x01020304, 5050)

	d.onLeaderChange(mi)
	leader := <-d.newLeader
	assert.Equal(t, *leader, "1.2.3.4:5050")
}

func TestOnLeaderChangeHostname(t *testing.T) {
	host := "2.3.4.5"
	d := NewZkLeaderDetector()
	mi := util.NewMasterInfo("id", 0x01020304, 5050)
	mi.Hostname = &host

	d.onLeaderChange(mi)
	leader := <-d.newLeader
	assert.Equal(t, *leader, "2.3.4.5:5050")
}
