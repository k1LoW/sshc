package sshc

import (
	"testing"
)

func TestUser(t *testing.T) {
	c, err := NewConfig("example.com", User("alice"))
	if err != nil {
		t.Fatal(err)
	}
	want := "alice"
	if got := c.user; got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}

func TestPort(t *testing.T) {
	c, err := NewConfig("example.com", Port(10022))
	if err != nil {
		t.Fatal(err)
	}
	want := 10022
	if got := c.port; got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}

func TestConfigPath(t *testing.T) {
	c, err := NewConfig("example.com", ConfigPath("/path/to/ssh_config"))
	if err != nil {
		t.Fatal(err)
	}

	want := 3
	if got := len(c.configPaths); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	want2 := "/path/to/ssh_config"
	if got := c.configPaths[0]; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}
