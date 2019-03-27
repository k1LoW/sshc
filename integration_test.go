package sshc

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
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

	err := startOpenSSHAgent(t)
	if err != nil {
		t.Fatal(err)
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

	err = addTestKey(t)
	if err != nil {
		t.Fatal(err)
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

	got, err := getHostname("server_with_ssh_agent")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%s\n", "server")
	if got != want {
		t.Fatalf("want = %#v, got = %#v", want, got)
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

func startOpenSSHAgent(t *testing.T) error {
	bin, err := exec.LookPath("ssh-agent")
	if err != nil {
		t.Fatalf("Could not find ssh-agent")
	}

	cmd := exec.Command(bin, "-s")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("cmd.Output: %v", err)
	}

	fields := bytes.Split(out, []byte(";"))
	path := bytes.SplitN(fields[0], []byte("="), 2)

	return os.Setenv("SSH_AUTH_SOCK", string(path[1]))
}

func addTestKey(t *testing.T) error {
	bin, err := exec.LookPath("ssh-add")
	if err != nil {
		t.Fatalf("could not find ssh-add")
	}

	cmd := exec.Command(bin, "./testdata/id_rsa")
	_, err = cmd.Output()
	if err != nil {
		t.Fatalf("cmd.Output: %v", err)
	}

	return nil
}
