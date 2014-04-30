package registry

import "testing"

func TestSetVersion(t *testing.T) {
	sc := NewServiceConfig("foo", "")
	if sc.Version() != "" {
		t.Fail()
	}

	sc.SetVersion("1")
	if sc.Version() != "1" {
		t.Fail()
	}

	sc.SetVersion("2")
	if sc.Version() != "2" {
		t.Fail()
	}

	sc.SetVersion("")
	if sc.Version() != "" {
		t.Fail()
	}
}

func TestSetEnv(t *testing.T) {
	sc := NewServiceConfig("foo", "")
	if len(sc.Env()) != 0 {
		t.Fail()
	}

	sc.EnvSet("foo", "bar")
	if sc.EnvGet("foo") != "bar" {
		t.Fail()
	}
	if sc.Env()["foo"] != "bar" {
		t.Fail()
	}

	sc.EnvSet("foo", "baz")
	if sc.EnvGet("foo") != "baz" {
		t.Fail()
	}
	if sc.Env()["foo"] != "baz" {
		t.Fail()
	}

	sc.EnvSet("bing", "bang")
	if len(sc.Env()) != 2 {
		t.Fail()
	}
}

func TestPorts(t *testing.T) {
	sc := NewServiceConfig("foo", "")

	if len(sc.Ports()) != 0 {
		t.Fail()
	}

	sc.AddPort("8000", "tcp")
	if len(sc.Ports()) != 1 {
		t.Fail()
	}
	if sc.Ports()["8000"] != "tcp" {
		t.Fail()
	}

	sc.AddPort("9000", "udp")
	if len(sc.Ports()) != 2 {
		t.Fail()
	}
	if sc.Ports()["9000"] != "udp" {
		t.Fail()
	}

	sc.ClearPorts()
	if len(sc.Ports()) != 0 {
		t.Fail()
	}
}

func TestID(t *testing.T) {
	sc := NewServiceConfig("foo", "")
	id := sc.ID()
	if id != 0 {
		t.Fail()
	}

	sc.SetVersion("foo")
	if sc.ID() < id {
		t.Fail()
	}
	id = sc.ID()

	sc.AddPort("8000", "tcp")
	if sc.ID() < id {
		t.Fail()
	}
	id = sc.ID()

	sc.EnvSet("foo", "bar")
	if sc.ID() < id {
		t.Fail()
	}
}