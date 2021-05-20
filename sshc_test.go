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

func TestPassphrase(t *testing.T) {
	c, err := NewConfig("example.com", Passphrase([]byte("secret")))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("secret")
	if got := c.passphrase; string(got) != string(want) {
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

func TestAppendConfigPath(t *testing.T) {
	c, err := NewConfig("example.com", AppendConfigPath("/path/to/ssh_config"))
	if err != nil {
		t.Fatal(err)
	}

	want := 3
	if got := len(c.configPaths); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	want2 := "~/.ssh/config"
	if got := c.configPaths[0]; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}

func TestClearConfigPath(t *testing.T) {
	c, err := NewConfig("example.com", ClearConfigPath(), ConfigPath("/path/to/ssh_config"))
	if err != nil {
		t.Fatal(err)
	}

	want := 1
	if got := len(c.configPaths); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	want2 := "/path/to/ssh_config"
	if got := c.configPaths[0]; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}

func TestKnownhosts(t *testing.T) {
	c, err := NewConfig("example.com", Knownhosts("/path/to/.ssh/known_hosts", "/root/.ssh/known_hosts"))
	if err != nil {
		t.Fatal(err)
	}

	want := 2
	if got := len(c.knownhosts); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	want2 := "/path/to/.ssh/known_hosts"
	if got := c.knownhosts[0]; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}

func TestGet(t *testing.T) {
	host := "server"
	c, err := NewConfig(host, ClearConfigPath(), ConfigPath("./testdata/ssh_config"))
	if err != nil {
		t.Fatal(err)
	}

	want := "172.30.0.3"
	if got := c.Get(host, "Hostname"); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}
