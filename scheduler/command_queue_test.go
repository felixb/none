package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnqueueAddsId(t *testing.T) {
	cq := NewCommandQueue()
	c := &Command{}
	cq.Enqueue(c)

	assert.NotNil(t, cq)
	assert.NotEmpty(t, c.Id, "command should have got an id")
	assert.NotNil(t, cq.GetCommandById(c.Id))
	assert.Equal(t, c.Id, cq.GetCommandById(c.Id).Id)

	cmds := cq.GetCommands()
	assert.NotNil(t, cmds[c.Id])
	assert.Equal(t, c.Id, cmds[c.Id].Id)
}

func TestNext(t *testing.T) {
	cq := NewCommandQueue()
	c0 := &Command{}
	cq.Enqueue(c0)

	c := cq.Next()
	assert.NotNil(t, c)
	assert.Equal(t, c0.Id, c.Id)

	assert.NotNil(t, cq.GetCommand())
	assert.Equal(t, c0.Id, cq.GetCommand().Id)
}

func TestClosed(t *testing.T) {
	cq := NewCommandQueue()
	c := &Command{}
	cq.Enqueue(c)
	assert.False(t, cq.Closed())

	cq.Close()
	assert.False(t, cq.Closed())

	assert.NotNil(t, cq.Next())
	assert.False(t, cq.Closed())

	assert.Nil(t, cq.Next())
	assert.True(t, cq.Closed())
}
