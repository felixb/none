package main

import (
	"testing"

	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/stretchr/testify/assert"
)

func TestFilterOfferMatch(t *testing.T) {
	o := &mesos.Offer{}
	c := new(MockConstraint)
	c.On("Match", o).Return(true)
	rf := ResourceFilter{Constraints: c}

	assert.True(t, rf.FilterOffer(o))
	c.AssertExpectations(t)
}

func TestFilterOfferNoMatch(t *testing.T) {
	o := &mesos.Offer{}
	c := new(MockConstraint)
	c.On("Match", o).Return(false)
	rf := ResourceFilter{Constraints: c}

	assert.False(t, rf.FilterOffer(o))
	c.AssertExpectations(t)
}

func TestFilterResources(t *testing.T) {
	rf := ResourceFilter{}
	o := util.NewOffer(util.NewOfferID("offerid"), util.NewFrameworkID("frameworkid"), util.NewSlaveID("slaveId"), "hostname")
	o.Resources = []*mesos.Resource{
		util.NewScalarResource("name", 1.0),
		util.NewScalarResource("ub0r-resource", 2.0),
		util.NewScalarResource("ub0r-resource", 3.0),
	}

	res := rf.FilterResources(o, "ub0r-resource")

	assert.Equal(t, 2, len(res))
	assert.Equal(t, "ub0r-resource", res[0].GetName())
}

func TestFilterRoleNil(t *testing.T) {
	rf := ResourceFilter{}

	assert.True(t, rf.filterRole("*"))
	assert.False(t, rf.filterRole("foo"))
}

func TestFilterRoleNotNil(t *testing.T) {
	r := "foo"
	rf := ResourceFilter{Role: &r}

	assert.True(t, rf.filterRole("*"))
	assert.True(t, rf.filterRole("foo"))
	assert.False(t, rf.filterRole("bar"))
}

func TestSumScalarResources(t *testing.T) {
	res := []*mesos.Resource{
		util.NewScalarResource("foo", 1.0),
		util.NewScalarResource("foo", 2.0),
	}

	assert.Equal(t, 3.0, SumScalarResources(res))
}
