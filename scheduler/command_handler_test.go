package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCommandHandler(t *testing.T) {
	ch := NewCommandHandler()
	assert.NotNil(t, ch)
}

func TestHasFailures(t *testing.T) {
	ch := NewCommandHandler()
	assert.False(t, ch.HasFailures())

	c := &Command{}
	ch.CommandLaunched(c)
	assert.False(t, ch.HasFailures())

	ch.CommandEnded(c)
	ch.CommandFinished(c)
	assert.False(t, ch.HasFailures())

	c = &Command{}
	ch.CommandLaunched(c)
	assert.False(t, ch.HasFailures())

	ch.CommandEnded(c)
	ch.CommandFailed(c)
	assert.True(t, ch.HasFailures())
}

func TestHasRunningTasks(t *testing.T) {
	ch := NewCommandHandler()
	assert.False(t, ch.HasRunningTasks())

	c := &Command{}
	ch.CommandLaunched(c)
	assert.True(t, ch.HasRunningTasks())

	ch.CommandEnded(c)
	assert.False(t, ch.HasRunningTasks())

	c = &Command{}
	ch.CommandLaunched(c)
	assert.True(t, ch.HasRunningTasks())

	ch.CommandEnded(c)
	assert.False(t, ch.HasRunningTasks())
}
