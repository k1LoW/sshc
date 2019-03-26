package sshc

import (
	"bytes"
	"flag"
	"fmt"
	"testing"
)

var (
	integration = flag.Bool("integration", false, "run integration tests")
)

var sshTests = []struct {
	hostname string
}{
	{"bastion"},
	{"server"},
}

func TestSSH(t *testing.T) {
	if !*integration {
		t.Skip()
	}
	for _, tt := range sshTests {
		got, err := getHostname(tt.hostname)
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("%s\n", tt.hostname)
		if got != want {
			t.Fatalf("want = %#v, got = %#v", want, got)
		}
	}
}

func getHostname(dest string) (string, error) {
	client, err := NewClient(dest, ConfigPath("./testdata/ssh_config"))
	if err != nil {
		return "", err
	}

	session, _ := client.NewSession()
	defer session.Close()

	var stdout = &bytes.Buffer{}
	session.Stdout = stdout
	err = session.Run("hostname")
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}
