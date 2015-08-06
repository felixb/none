package main

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
)

type ResourceFilter struct {
	Role        *string
	Constraints Constraint
}

func (rf *ResourceFilter) FilterOffer(offer *mesos.Offer) bool {
	return rf.Constraints.Match(offer)
}

func (rf *ResourceFilter) FilterResources(offer *mesos.Offer, name string) []*mesos.Resource {
	return util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
		return res.GetName() == name && rf.filterRole(res.GetRole())
	})
}

func (rf *ResourceFilter) filterRole(role string) bool {
	return role == "*" || rf.Role != nil && role == *rf.Role
}

func SumScalarResources(res []*mesos.Resource) float64 {
	v := 0.0
	for _, res := range res {
		v += res.GetScalar().GetValue()
	}
	return v
}
