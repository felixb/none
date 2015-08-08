package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockStringWriter struct {
	LastString string
	Writes     int
}

func (m *MockStringWriter) WriteString(s string) (int, error) {
	m.LastString = s
	m.Writes++
	return 0, nil
}

func TestUpdate(t *testing.T) {
	m := &MockStringWriter{}
	p := &Pailer{writer: m}

	u := &update{
		Offset: 0,
		Data:   "fooo",
	}

	p.update(u)
	assert.Equal(t, 4, p.Offset)
	assert.Equal(t, "fooo", m.LastString)
	assert.Equal(t, 1, m.Writes)

	u = &update{
		Offset: 4,
		Data:   "bar",
	}

	p.update(u)
	assert.Equal(t, 7, p.Offset)
	assert.Equal(t, "bar", m.LastString)
	assert.Equal(t, 2, m.Writes)
}
