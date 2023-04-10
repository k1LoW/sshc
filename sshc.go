// Package sshc provides sshc.NewClient() that returns *ssh.Client using ssh_config(5)
package sshc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ScaleFT/sshkeys"
	"github.com/k1LoW/exec"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

type KeyAndPassphrase struct {
	key        []byte
	passphrase []byte
}

type DialConfig struct {
	Hostname          string
	User              string
	Port              int
	UseAgent          bool
	Knownhosts        []string
	KeyAndPassphrases []KeyAndPassphrase
	ProxyCommand      string
	ProxyJump         string
	Password          string
	Timeout           time.Duration
	Wd                string
	Auth              []ssh.AuthMethod
}

// NewClient reads ssh_config(5) ( Default is ~/.ssh/config and /etc/ssh/ssh_config ) and returns *ssh.Client.
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	c, err := NewConfig(options...)
	if err != nil {
		return nil, err
	}
	pc, wd := c.getProxyCommand(host)
	if wd == "" {
		wd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	dc := &DialConfig{
		User:         c.getUser(host),
		ProxyCommand: pc,
		ProxyJump:    c.Get(host, "ProxyJump"),
		Knownhosts:   c.knownhosts,
		UseAgent:     c.useAgent,
		Password:     c.password,
		Wd:           wd,
		Auth:         c.auth,
	}
	hostname, err := c.getHostname(host)
	if err != nil {
		return nil, err
	}
	dc.Hostname = hostname
	port, err := c.getPort(host)
	if err != nil {
		return nil, err
	}
	dc.Port = port
	keys, err := c.getKeyAndPassphrases(host)
	if err != nil {
		return nil, err
	}
	dc.KeyAndPassphrases = keys

	return Dial(dc)
}

// Dial returns *ssh.Client using Config
func Dial(dc *DialConfig) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", dc.Hostname, dc.Port)
	var (
		signers []ssh.Signer
		err     error
	)
	auth := []ssh.AuthMethod{}
	for _, k := range dc.KeyAndPassphrases {
		signer, err := sshkeys.ParseEncryptedPrivateKey(k.key, k.passphrase)
		if err != nil {
			// passphrase
			fmt.Print("Enter passphrase for key: ")
			passPhrase, err := term.ReadPassword(0)
			if err != nil {
				fmt.Println("")
				return nil, err
			}
			signer, err = sshkeys.ParseEncryptedPrivateKey(k.key, passPhrase)
			if err != nil {
				fmt.Println("")
				return nil, err
			}
			fmt.Println("")
		}
		signers = append(signers, signer)
	}
	useAgent := false
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
			useAgent = true
		}
	}
	if len(signers) > 0 && !useAgent {
		auth = append(auth, ssh.PublicKeys(signers...))
	}

	// password
	if dc.Password != "" {
		auth = append(auth, ssh.Password(dc.Password))
	}

	// additional ssh.AuthMethod
	auth = append(auth, dc.Auth...)

	cb, err := hostKeyCallback(dc.Knownhosts)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            dc.User,
		Auth:            auth,
		HostKeyCallback: cb,
		Timeout:         dc.Timeout,
	}

	proxyCommand := dc.ProxyCommand
	if proxyCommand == "" && dc.ProxyJump != "" {
		parsedProxyJump, err := parseProxyJump(dc.ProxyJump)
		if err != nil {
			return nil, err
		}
		proxyCommand = expandVerbs(parsedProxyJump, dc.User, dc.Port, dc.Hostname)
	}

	if proxyCommand != "" {
		client, server := net.Pipe()
		unescapedProxyCommand := expandVerbs(proxyCommand, dc.User, dc.Port, dc.Hostname)
		cmd := exec.Command("sh", "-c", unescapedProxyCommand) // #nosec
		cmd.Dir = dc.Wd
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
				if err := exec.KillCommand(cmd); err != nil {
					return nil, err
				}
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

func expandVerbs(v, user string, port int, hostname string) string {
	v = strings.Replace(v, "%h", hostname, -1)
	v = strings.Replace(v, "%p", strconv.Itoa(port), -1)
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
