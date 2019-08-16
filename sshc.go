// Package sshc provides sshc.NewClient() that returns *ssh.Client using ssh_config(5)
package sshc

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ScaleFT/sshkeys"
	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
)

var defaultConfigPaths = []string{
	filepath.Join("~", ".ssh", "config"),
	filepath.Join("/", "etc", "ssh", "ssh_config"),
}

// Config is the type for the SSH Client config. not ssh_config.
type Config struct {
	configPaths []string
	host        string
	user        string
	port        int
	passphrase  []byte
	useAgent    bool
	configs     []*ssh_config.Config
	loader      sync.Once
	knownhosts  []string
}

// NewConfig creates SSH client config.
func NewConfig(host string, options ...Option) (*Config, error) {
	var err error

	c := &Config{
		configPaths: defaultConfigPaths,
		host:        host,
		useAgent:    true, // Default is true
	}
	for _, option := range options {
		err = option(c)
		if err != nil {
			return nil, err
		}
	}

	c.user = c.Get(host, "User")
	c.port, err = strconv.Atoi(c.Get(host, "Port"))
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		err = option(c)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NewClient reads ssh_config(5) ( Default is ~/.ssh/config and /etc/ssh/ssh_config ) and returns *ssh.Client.
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	c, err := NewConfig(host, options...)
	if err != nil {
		return nil, err
	}
	return c.DialWithConfig()
}

// Get returns Config value.
func (c *Config) Get(alias, key string) string {
	homeDir, err := homedir.Dir()
	if err != nil {
		return ""
	}
	c.loader.Do(func() {
		for _, p := range c.configPaths {
			cPath := strings.Replace(p, "~", homeDir, 1)
			if _, err := os.Lstat(cPath); err != nil {
				continue
			}
			f, err := os.Open(filepath.Clean(cPath))
			if err != nil {
				continue
			}
			cfg, err := ssh_config.Decode(f)
			if err != nil {
				continue
			}
			c.configs = append([]*ssh_config.Config{cfg}, c.configs...)
		}
	})
	for _, cfg := range c.configs {
		val, err := cfg.Get(alias, key)
		if err != nil || val != "" {
			return val
		}
	}
	return ssh_config.Default(key)
}

// DialWithConfig returns *ssh.Client using Config
func (c *Config) DialWithConfig() (*ssh.Client, error) {
	host := c.host
	user := c.user
	port := strconv.Itoa(c.port)
	hostname, err := c.getHostname()
	if err != nil {
		return nil, err
	}
	addr := hostname + ":" + port

	auth := []ssh.AuthMethod{}
	keyPath, err := c.getIdentityFile()
	if err != nil {
		return nil, err
	}
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return nil, err
	}
	signer, err := sshkeys.ParseEncryptedPrivateKey(key, c.passphrase)
	if err != nil {
		// passphrase
		fmt.Printf("Enter passphrase for key '%s': ", keyPath)
		passPhrase, err := terminal.ReadPassword(0)
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

	if c.useAgent && sshAuthSockExists() {
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
		} else {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	} else {
		auth = append(auth, ssh.PublicKeys(signer))
	}

	cb, err := c.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: cb,
	}

	proxyCommand := c.Get(host, "ProxyCommand")
	if proxyCommand != "" {
		client, server := net.Pipe()
		proxyCommand = c.unescapeCharacters(proxyCommand)
		proxyCommand = strings.Replace(proxyCommand, "%p", port, -1)
		proxyCommand = strings.Replace(proxyCommand, "%r", user, -1)
		cmd := exec.Command("sh", "-c", proxyCommand) // #nosec
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = server
		cmd.Stdout = server
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return nil, err
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

func (c *Config) unescapeCharacters(v string) string {
	user := c.user
	port := strconv.Itoa(c.port)
	hostname, _ := c.getHostname()
	v = strings.Replace(v, "%h", hostname, -1)
	v = strings.Replace(v, "%p", port, -1)
	v = strings.Replace(v, "%r", user, -1)
	return v
}

func (c *Config) getHostname() (string, error) {
	h := c.Get(c.host, "Hostname")
	if h == "" {
		return c.host, nil
	}
	return h, nil
}

func (c *Config) getIdentityFile() (string, error) {
	keyPath := c.Get(c.host, "IdentityFile")
	keyPath = c.unescapeCharacters(keyPath)
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	if keyPath == "~/.ssh/identity" {
		if _, err := os.Lstat(strings.Replace(keyPath, "~", homeDir, 1)); err != nil {
			keyPath = "~/.ssh/id_rsa"
		}
	}
	return strings.Replace(keyPath, "~", homeDir, 1), nil
}

func (c *Config) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if len(c.knownhosts) > 0 {
		hostKeyCallback, err := knownhosts.New(c.knownhosts...)
		if err != nil {
			return nil, err
		}
		return hostKeyCallback, nil
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}, nil
}

func unique(s []string) []string {
	keys := make(map[string]bool)
	l := []string{}
	for _, e := range s {
		if _, v := keys[e]; !v {
			keys[e] = true
			l = append(l, e)
		}
	}
	return l
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
