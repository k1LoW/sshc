package sshc

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var integration = flag.Bool("integration", false, "run integration tests")

func TestSSH(t *testing.T) {
	if !*integration {
		t.Skip()
	}

	sshTests := []struct {
		hostname string
	}{
		{"bastion"},
		{"server"},
	}

	t.Run("SSH connection test without ssh-agent", func(t *testing.T) {
		for _, tt := range sshTests {
			got, err := getHostname(tt.hostname, false, false)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf("%s\n", tt.hostname)
			if got != want {
				t.Fatalf("want = %#v, got = %#v", want, got)
			}
		}
	})

	t.Run("SSH connection test using sudo", func(t *testing.T) {
		for _, tt := range sshTests {
			got, err := getHostname(tt.hostname, false, true)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf("%s\n", tt.hostname)
			if got != want {
				t.Fatalf("want = %#v, got = %#v", want, got)
			}
		}
	})

	t.Run("SSH connection test using ssh-agent", func(t *testing.T) {
		err := startOpenSSHAgent(t)
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range sshTests {
			got, err := getHostname(tt.hostname, true, false)
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
			got, err := getHostname(tt.hostname, true, false)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf("%s\n", tt.hostname)
			if got != want {
				t.Fatalf("want = %#v, got = %#v", want, got)
			}
		}

		err = killOpenSSHAgent(t)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func getHostname(dest string, useAgent bool, sudo bool) (string, error) {
	client, err := NewClient(dest, ConfigPath("./testdata/ssh_config"), UseAgent(useAgent))
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout = &bytes.Buffer{}
	session.Stdout = stdout
	cmd := "hostname"

	if sudo {
		var stdin = &bytes.Buffer{}
		session.Stdin = stdin
		stdin.WriteString("k1low\n")
		cmd = "sudo -S hostname"
	}
	err = session.Run(cmd)
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
	pid := bytes.SplitN(fields[2], []byte("="), 2)
	err = os.Setenv("SSH_AUTH_SOCK", string(path[1]))
	if err != nil {
		return err
	}
	return os.Setenv("SSH_AGENT_PID", string(pid[1]))
}

func killOpenSSHAgent(t *testing.T) error {
	bin, err := exec.LookPath("ssh-agent")
	if err != nil {
		t.Fatalf("Could not find ssh-agent")
	}
	cmd := exec.Command(bin, "-k")
	_, err = cmd.Output()
	if err != nil {
		t.Fatalf("cmd.Output: %v", err)
	}
	return err
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
