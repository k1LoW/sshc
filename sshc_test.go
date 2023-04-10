package sshc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUser(t *testing.T) {
	c, err := NewConfig(User("alice"))
	if err != nil {
		t.Fatal(err)
	}
	want := "alice"
	if got := c.user; got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}

func TestPort(t *testing.T) {
	c, err := NewConfig(Port(10022))
	if err != nil {
		t.Fatal(err)
	}
	want := 10022
	if got := c.port; got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	{
		want := 12345
		c, err := NewConfig(ClearConfig(), Port(want))
		if err != nil {
			t.Fatal(err)
		}

		if got := c.port; got != want {
			t.Fatalf("want = %#v, got = %#v", want, got)
		}
	}
}

func TestHostname(t *testing.T) {
	c, err := NewConfig(Hostname("example.com"))
	if err != nil {
		t.Fatal(err)
	}
	want := "example.com"
	if got, _ := c.getHostname("dummy"); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}

func TestPassphrase(t *testing.T) {
	c, err := NewConfig(Passphrase([]byte("secret")))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("secret")
	if got := c.passphrase; string(got) != string(want) {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}
}

func TestConfigPath(t *testing.T) {
	c, err := NewConfig(ConfigPath("./testdata/simple/.ssh/config"))
	if err != nil {
		t.Fatal(err)
	}
	want := 1
	base, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range defaultConfigPaths {
		p, err := expandPath(p, base)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(p); err == nil {
			want++
		}
	}
	if got := len(c.configs); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want2 := filepath.Join(wd, "./testdata/simple/.ssh/config")
	if got := c.configs[0].path; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}

func TestAppendConfigPath(t *testing.T) {
	c, err := NewConfig(AppendConfigPath("./testdata/simple/.ssh/config"))
	if err != nil {
		t.Fatal(err)
	}
	want := 1
	base, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range defaultConfigPaths {
		p, err := expandPath(p, base)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(p); err == nil {
			want++
		}
	}
	if got := len(c.configs); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	if want > 1 {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		notwant := filepath.Join(wd, "./testdata/simple/.ssh/config")
		if got := c.configs[0].path; got == notwant {
			t.Fatalf("got = %#v", got)
		}
	}
}

func TestClearConfig(t *testing.T) {
	c, err := NewConfig(ClearConfig(), ConfigPath("./testdata/simple/.ssh/config"))
	if err != nil {
		t.Fatal(err)
	}

	want := 1
	if got := len(c.configs); got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want2 := filepath.Join(wd, "./testdata/simple/.ssh/config")
	if got := c.configs[0].path; got != want2 {
		t.Fatalf("want = %#v, got = %#v", want2, got)
	}
}

func TestKnownhosts(t *testing.T) {
	c, err := NewConfig(Knownhosts("/path/to/.ssh/known_hosts", "/root/.ssh/known_hosts"))
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
	homes := []string{
		"simple",
		"separate",
	}
	for _, h := range homes {
		t.Setenv("HOME", testHome(t, h))
		p := filepath.Join("./testdata", h, ".ssh", "config")
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
				"~/.ssh/id_rsa",
			},
			{
				"bastion",
				"127.0.0.1",
				"9022",
				"k1low",
				"~/.ssh/id_rsa",
			},
			{
				"simple",
				"simple",
				"22",
				"",
				"~/.ssh/identity",
			},
		}
		for _, tt := range tests {
			t.Run(tt.host, func(t *testing.T) {
				c, err := NewConfig(ClearConfig(), ConfigPath(p))
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
			})
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
			got, err := parseProxyJump(tt.text)
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

func testHome(t *testing.T, path string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(wd, "testdata", path))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
