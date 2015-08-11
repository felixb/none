package main

import (
	"fmt"
	"strings"

	mesos "github.com/mesos/mesos-go/mesosproto"
)

const (
	CONSTRAINT_OPERATOR_EQUALS = "EQUALS"
)

// Constraints

type Constraints []Constraint

func ParseConstraints(params *string) (Constraints, error) {
	if params == nil || len(*params) == 0 {
		return Constraints{}, nil
	}
	constraints := strings.Split(*params, ";")
	cs := make([]Constraint, len(constraints))
	for i, p := range constraints {
		c, err := ParseConstraint(&p)
		if err != nil {
			return nil, err
		}
		cs[i] = c
	}
	return cs, nil
}

func (cs Constraints) Match(offer *mesos.Offer) bool {
	for _, c := range cs {
		if !c.Match(offer) {
			return false
		}
	}
	return true
}

// Constraint

type Constraint interface {
	Match(offer *mesos.Offer) bool
}

func ParseConstraint(params *string) (Constraint, error) {
	if params == nil {
		return nil, fmt.Errorf("Params must not be nil")
	}
	parts := strings.SplitN(*params, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("Invalid Constraint definition: %s", *params)
	}
	if len(parts) == 2 {
		return NewConstraint(parts[0], parts[1], "")
	} else {
		return NewConstraint(parts[0], parts[1], parts[2])
	}
}

func NewConstraint(attr, operator, value string) (Constraint, error) {
	if operator == CONSTRAINT_OPERATOR_EQUALS {
		return NewEqualsConstraint(attr, value), nil
	}
	return nil, fmt.Errorf("Unsupported operator: %s", operator)
}

// EqualsConstraint

type EqualsConstraint struct {
	Attribute string
	Value     string
}

func NewEqualsConstraint(attr, value string) Constraint {
	return &EqualsConstraint{
		Attribute: attr,
		Value:     value,
	}
}

func (c *EqualsConstraint) Match(offer *mesos.Offer) bool {
	for _, a := range offer.GetAttributes() {
		if c.Attribute == a.GetName() {
			if a.GetType() == mesos.Value_TEXT {
				return c.Value == a.GetText().GetValue()
			} else if a.GetType() == mesos.Value_SCALAR {
				return c.Value == fmt.Sprintf("%.f", a.GetScalar().GetValue())
			} else {
				return false
			}
		}
	}
	return false
}

func (c *EqualsConstraint) String() string {
	return fmt.Sprintf("%s:%s:%s", c.Attribute, CONSTRAINT_OPERATOR_EQUALS, c.Value)
}
