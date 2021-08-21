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

	{
		want := 12345
		host := "simple"
		c, err := NewConfig(host, ClearConfigPath(), ConfigPath("./testdata/simple/ssh_config"), Port(want))
		if err != nil {
			t.Fatal(err)
		}

		if got := c.port; got != want {
			t.Fatalf("want = %#v, got = %#v", want, got)
		}
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
	tests := []struct {
		host             string
		wantHostname     string
		wantPort         string
		wantUser         string
		wantIdentityFile string
	}{
		{
			"server",
			"172.30.0.3",
			"22",
			"root",
			"./testdata/id_rsa",
		},
		{
			"bastion",
			"127.0.0.1",
			"9022",
			"k1low",
			"./testdata/id_rsa",
		},
	}
	for _, tt := range tests {
		c, err := NewConfig(tt.host, ClearConfigPath(), ConfigPath("./testdata/simple/ssh_config"))
		if err != nil {
			t.Fatal(err)
		}
		if got := c.Get(tt.host, "Hostname"); got != tt.wantHostname {
			t.Errorf("want = %#v, got = %#v", tt.wantHostname, got)
		}
		if got := c.Get(tt.host, "Port"); got != tt.wantPort {
			t.Errorf("want = %#v, got = %#v", tt.wantPort, got)
		}
		if got := c.Get(tt.host, "User"); got != tt.wantUser {
			t.Errorf("want = %#v, got = %#v", tt.wantUser, got)
		}
		if got := c.Get(tt.host, "IdentityFile"); got != tt.wantIdentityFile {
			t.Errorf("want = %#v, got = %#v", tt.wantIdentityFile, got)
		}
	}
}

func TestConfig_parseProxyJump(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name:    "full config",
			text:    "user@host:2002",
			want:    "ssh -l %r -W %h:%p  user@host -p 2002",
			wantErr: false,
		},
		{
			name:    "not defined port",
			text:    "user@host",
			want:    "ssh -l %r -W %h:%p  user@host -p 22",
			wantErr: false,
		},
		{
			name:    "not defined user",
			text:    "host:2222",
			want:    "ssh -l %r -W %h:%p  host -p 2222",
			wantErr: false,
		},
		{
			name:    "wrong port format",
			text:    "user@host:xxxxx",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewConfig("example.com")
			got, err := c.parseProxyJump(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.parseProxyJump() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.parseProxyJump() = %v, want %v", got, tt.want)
			}
		})
	}
}
