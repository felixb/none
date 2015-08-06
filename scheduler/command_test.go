package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesResources(t *testing.T) {
	c := &Command{
		CpuReq: 1,
		MemReq: 128,
	}

	assert.False(t, c.MatchesResources(0.1, 0.1))
	assert.False(t, c.MatchesResources(1000, 0.1))
	assert.False(t, c.MatchesResources(0.1, 1000))
	assert.True(t, c.MatchesResources(1, 128))
	assert.True(t, c.MatchesResources(2, 256))
}

func TestGetCommandInfo(t *testing.T) {
	c := &Command{
		Cmd: "foo",
	}

	ci := c.GetCommandInfo()
	assert.NotNil(t, ci)
	assert.Equal(t, "sh", ci.GetValue())
	assert.False(t, ci.GetShell())
}

func TestGetResources(t *testing.T) {
	c := &Command{
		CpuReq: 2,
		MemReq: 256,
	}

	res := c.GetResources()
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "cpus", res[0].GetName())
	assert.Equal(t, 2.0, res[0].GetScalar().GetValue())
	assert.Equal(t, "mem", res[1].GetName())
	assert.Equal(t, 256.0, res[1].GetScalar().GetValue())
}
