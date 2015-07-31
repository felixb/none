package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMasterState(t *testing.T) {
	f, err := os.Open("../fixtures/master_state.json")
	assert.Nil(t, err, "Unexpected error")
	defer f.Close()

	s, err := NewMasterState(f)
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, s, "Expected state")

	assert.Equal(t, 1, len(s.Slaves), "Expected 1 slave")

	slv := s.Slaves[0]
	assert.Equal(t, "http://10.141.141.10:5051/slave(1)/state.json", slv.GetStateUrl(), "slave state url")
}

func TestNewSlaveState(t *testing.T) {
	f, err := os.Open("../fixtures/slave_state.json")
	assert.Nil(t, err, "Unexpected error")
	defer f.Close()

	s, err := NewSlaveState(f)
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, s, "Expected state")

	assert.Equal(t, 1, len(s.Frameworks), "Expected 1 framework")
}

func TestGetExecutor(t *testing.T) {
	f, err := os.Open("../fixtures/slave_state.json")
	assert.Nil(t, err, "Unexpected error")
	defer f.Close()
	s, err := NewSlaveState(f)
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, s, "Expected state")
	fid := "20150730-183810-177048842-5050-1233-0001"
	tid := "1"

	assert.NotNil(t, s.GetFramework(fid), "Framework not found")
	assert.NotNil(t, s.GetFramework(fid).GetExecutor(tid), "Task not found")
}

func TestGetDirectory(t *testing.T) {
	f, err := os.Open("../fixtures/slave_state.json")
	assert.Nil(t, err, "Unexpected error")
	defer f.Close()
	s, err := NewSlaveState(f)
	assert.Nil(t, err, "Unexpected error")
	assert.NotNil(t, s, "Expected state")

	fid := "20150730-183810-177048842-5050-1233-0001"
	tid := "1"
	d := "/tmp/mesos/slaves/20150730-183810-177048842-5050-1233-S0/frameworks/20150730-183810-177048842-5050-1233-0001/executors/1/runs/40a724b1-34f0-417b-a601-89a0a8314c0c"

	v := s.GetDirectory(fid, tid)
	assert.NotNil(t, v, "Directory not found")
	assert.Equal(t, d, *v, "Directory mismatch")
}
