// Package sshc provides sshc.NewClient() that returns *ssh.Client using ssh_config(5)
package sshc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ScaleFT/sshkeys"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

type DialConfig struct {
	Hostname     string
	User         string
	Port         int
	Passphrase   []byte
	UseAgent     bool
	Knownhosts   []string
	IdentityFile string
	ProxyCommand string
	ProxyJump    string
	Password     string
}

// NewClient reads ssh_config(5) ( Default is ~/.ssh/config and /etc/ssh/ssh_config ) and returns *ssh.Client.
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	c, err := NewConfig(options...)
	if err != nil {
		return nil, err
	}
	dc := &DialConfig{
		User:         c.Get(host, "User"),
		ProxyCommand: c.Get(host, "ProxyCommand"),
		ProxyJump:    c.Get(host, "ProxyJump"),
		Passphrase:   c.passphrase,
		Knownhosts:   c.knownhosts,
		UseAgent:     c.useAgent,
		Password:     c.password,
	}
	hostname, err := c.getHostname(host)
	if err != nil {
		return nil, err
	}
	dc.Hostname = hostname
	port, err := strconv.Atoi(c.Get(host, "Port"))
	if err != nil {
		return nil, err
	}
	dc.Port = port
	keyPath, err := c.getIdentityFile(host)
	if err != nil {
		return nil, err
	}
	dc.IdentityFile = keyPath
	fmt.Printf("%#v\n", dc)

	return Dial(dc)
}

// Dial returns *ssh.Client using Config
func Dial(dc *DialConfig) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", dc.Hostname, dc.Port)

	var signer ssh.Signer
	auth := []ssh.AuthMethod{}
	if dc.IdentityFile != "" {
		key, err := os.ReadFile(filepath.Clean(dc.IdentityFile))
		if err != nil {
			return nil, err
		}
		signer, err = sshkeys.ParseEncryptedPrivateKey(key, dc.Passphrase)
		if err != nil {
			// passphrase
			fmt.Printf("Enter passphrase for key '%s': ", dc.IdentityFile)
			passPhrase, err := term.ReadPassword(0)
			if err != nil {
				fmt.Println("")
				return nil, err
			}
			signer, err = sshkeys.ParseEncryptedPrivateKey(key, passPhrase)
			if err != nil {
				fmt.Println("")
				return nil, err
			}
			fmt.Println("")
		}
	}
	if dc.UseAgent && sshAuthSockExists() {
		sshAgentClient, err := newSSHAgentClient()
		if err != nil {
			return nil, err
		}
		identities, err := sshAgentClient.List()
		if err != nil {
			return nil, err
		}
		if len(identities) > 0 {
			auth = append(auth, ssh.PublicKeysCallback(sshAgentClient.Signers))
		} else if signer != nil {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	} else if signer != nil {
		auth = append(auth, ssh.PublicKeys(signer))
	}

	// password
	if dc.Password != "" {
		auth = append(auth, ssh.Password(dc.Password))
	}

	cb, err := hostKeyCallback(dc.Knownhosts)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            dc.User,
		Auth:            auth,
		HostKeyCallback: cb,
	}

	proxyCommand := dc.ProxyCommand
	if proxyCommand == "" && dc.ProxyJump != "" {
		parsedProxyJump, err := parseProxyJump(dc.ProxyJump)
		if err != nil {
			return nil, err
		}
		proxyCommand = unescapeCharacters(parsedProxyJump, dc.User, strconv.Itoa(dc.Port), dc.Hostname)
	}

	if proxyCommand != "" {
		client, server := net.Pipe()
		unescapedProxyCommand := unescapeCharacters(proxyCommand, dc.User, strconv.Itoa(dc.Port), dc.Hostname)
		cmd := exec.Command("sh", "-c", unescapedProxyCommand) // #nosec
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = server
		cmd.Stdout = server
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("proxy command:%s error:%s", unescapedProxyCommand, err)
		}

		done := make(chan *ssh.Client)
		errchan := make(chan error)
		go func() {
			conn, incomingChannels, incomingRequests, err := ssh.NewClientConn(client, addr, sshConfig)
			if err != nil {
				errchan <- err
				return
			}
			done <- ssh.NewClient(conn, incomingChannels, incomingRequests)
		}()

		for {
			select {
			case err := <-errchan:
				return nil, err
			case <-time.After(30 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				return nil, fmt.Errorf("proxy command timeout(30sec)")
			case client := <-done:
				return client, nil
			}
		}

	}
	return ssh.Dial("tcp", addr, sshConfig)
}

func newSSHAgentClient() (agent.ExtendedAgent, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}

	return agent.NewClient(conn), nil
}

func sshAuthSockExists() bool {
	return os.Getenv("SSH_AUTH_SOCK") != ""
}

func parseProxyJump(text string) (string, error) {
	proxyPort := "22"
	if strings.Contains(text, ":") {
		var portReg = regexp.MustCompile(`.+:(?P<port>[0-9]+)`)
		match := portReg.FindAllStringSubmatch(text, -1)
		if len(match) == 0 {
			return "", errors.New("proxyJump is wrong format")
		}

		for i, name := range portReg.SubexpNames() {
			if i != 0 && name == "port" {
				proxyPort = match[0][i]
			}
		}
		text = text[:strings.Index(text, ":")]
	}
	return fmt.Sprintf("ssh -l %%r -W %%h:%%p  %s -p %s", text, proxyPort), nil
}

func unescapeCharacters(v, user, port, hostname string) string {
	v = strings.Replace(v, "%h", hostname, -1)
	v = strings.Replace(v, "%p", port, -1)
	v = strings.Replace(v, "%r", user, -1)
	return v
}

func hostKeyCallback(files []string) (ssh.HostKeyCallback, error) {
	if len(files) > 0 {
		hostKeyCallback, err := knownhosts.New(files...)
		if err != nil {
			return nil, err
		}
		return hostKeyCallback, nil
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}, nil
}
