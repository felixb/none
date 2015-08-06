package main

import (
	"testing"

	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mocks

type MockConstraint struct {
	mock.Mock
}

func (c *MockConstraint) Match(offer *mesos.Offer) bool {
	args := c.Called(offer)
	return args.Bool(0)
}

// Constraints

func TestParseConstraints(t *testing.T) {
	p := "foo:EQUALS:bar;f00:EQUALS:BAR"
	cs, err := ParseConstraints(&p)
	assert.Nil(t, err)
	assert.NotNil(t, cs)
	assert.Equal(t, 2, len(cs), "should have 2 Constraints")

	assert.NotNil(t, cs[0])
	assert.NotNil(t, cs[1])
}

func TestParseConstraintsNil(t *testing.T) {
	cs, err := ParseConstraints(nil)
	assert.Nil(t, err)
	assert.NotNil(t, cs)
	assert.Equal(t, 0, len(cs), "should have 0 Constraints")
}

func TestMatchAllEmpty(t *testing.T) {
	cs := &Constraints{}

	o := &mesos.Offer{
		Attributes: nil,
	}

	assert.True(t, cs.Match(o), "empty constraints should always match")
}

func TestMatchAllTrue(t *testing.T) {
	o := &mesos.Offer{
		Attributes: nil,
	}

	tc := new(MockConstraint)
	tc.On("Match", o).Return(true)

	cs := &Constraints{tc}

	assert.True(t, cs.Match(o))
	tc.AssertExpectations(t)
}

func TestMatchAllFalse(t *testing.T) {
	o := &mesos.Offer{
		Attributes: nil,
	}

	fc := new(MockConstraint)
	fc.On("Match", o).Return(false)

	cs := &Constraints{fc}

	assert.False(t, cs.Match(o))
	fc.AssertExpectations(t)
}

func TestMatchMixed(t *testing.T) {
	o := &mesos.Offer{
		Attributes: nil,
	}

	tc := new(MockConstraint)
	tc.On("Match", o).Return(true)

	fc := new(MockConstraint)
	fc.On("Match", o).Return(false)

	cs := &Constraints{tc, fc}
	assert.False(t, cs.Match(o))
	tc.AssertExpectations(t)
	fc.AssertExpectations(t)
}

// Constraint

func TestParseConstraint(t *testing.T) {
	p := "foo:EQUALS:bar"
	c, err := ParseConstraint(&p)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.IsType(t, &EqualsConstraint{}, c)

	ec := c.(*EqualsConstraint)
	assert.Equal(t, "foo", ec.Attribute)
	assert.Equal(t, "bar", ec.Value)
}
func TestParseConstraintNil(t *testing.T) {
	c, err := ParseConstraint(nil)
	assert.NotNil(t, err)
	assert.Nil(t, c)
}

// EqualsConstraint

func TestMatchEqualsWithNilAttributes(t *testing.T) {
	c := NewEqualsConstraint("foo", "bar")

	o := &mesos.Offer{
		Attributes: nil,
	}

	assert.False(t, c.Match(o))
}

func TestMatchEqualsWithZeroAttributes(t *testing.T) {
	c := NewEqualsConstraint("foo", "bar")

	o := &mesos.Offer{
		Attributes: []*mesos.Attribute{},
	}

	assert.False(t, c.Match(o))
}

func TestMatchEqualsWithOtherAttributes(t *testing.T) {
	c := NewEqualsConstraint("foo", "bar")

	an := "f00"
	at := mesos.Value_TEXT
	av := "bar"

	o := &mesos.Offer{
		Attributes: []*mesos.Attribute{&mesos.Attribute{
			Name: &an,
			Type: &at,
			Text: &mesos.Value_Text{
				Value: &av,
			},
		}},
	}

	assert.False(t, c.Match(o))
}

func TestMatchEqualsWithOtherValue(t *testing.T) {
	c := NewEqualsConstraint("foo", "bar")

	an := "foo"
	at := mesos.Value_TEXT
	av := "BAR"

	o := &mesos.Offer{
		Attributes: []*mesos.Attribute{&mesos.Attribute{
			Name: &an,
			Type: &at,
			Text: &mesos.Value_Text{
				Value: &av,
			},
		}},
	}

	assert.False(t, c.Match(o))
}

func TestMatchEqualsMatchingText(t *testing.T) {
	c := NewEqualsConstraint("foo", "bar")

	an := "foo"
	at := mesos.Value_TEXT
	av := "bar"

	o := &mesos.Offer{
		Attributes: []*mesos.Attribute{&mesos.Attribute{
			Name: &an,
			Type: &at,
			Text: &mesos.Value_Text{
				Value: &av,
			},
		}},
	}

	assert.True(t, c.Match(o))
}

func TestMatchEqualsMatchingInt(t *testing.T) {
	c := NewEqualsConstraint("foo", "13")

	an := "foo"
	at := mesos.Value_SCALAR
	av := 13.0

	o := &mesos.Offer{
		Attributes: []*mesos.Attribute{&mesos.Attribute{
			Name: &an,
			Type: &at,
			Scalar: &mesos.Value_Scalar{
				Value: &av,
			},
		}},
	}

	assert.True(t, c.Match(o))
}
