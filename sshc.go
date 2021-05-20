// Package sshc provides sshc.NewClient() that returns *ssh.Client using ssh_config(5)
package sshc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ScaleFT/sshkeys"
	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
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
			f, err := os.Open(cPath)
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
	key, err := ioutil.ReadFile(keyPath)
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

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // FIXME
	}

	proxyCommand := c.Get(host, "ProxyCommand")
	proxyJump := c.Get(host, "ProxyJump")

	if proxyJump != "" {
		parsedProxyJump, err := c.parseProxyJump(proxyJump)
		if err != nil {
			return nil, err
		}
		proxyCommand = parsedProxyJump
	}

	if proxyCommand != "" {
		client, server := net.Pipe()
		proxyCommand = c.unescapeCharacters(proxyCommand)
		proxyCommand = strings.Replace(proxyCommand, "%p", port, -1)
		proxyCommand = strings.Replace(proxyCommand, "%r", user, -1)
		cmd := exec.Command("sh", "-c", proxyCommand)
		cmd.Stdin = server
		cmd.Stdout = server
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		conn, incomingChannels, incomingRequests, err := ssh.NewClientConn(client, addr, sshConfig)
		if err != nil {
			return nil, err
		}
		return ssh.NewClient(conn, incomingChannels, incomingRequests), nil
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

func (c *Config) parseProxyJump(text string) (string, error) {
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
	return fmt.Sprintf("ssh -l %%r -W %%h:%%p  %s -p %s", c.unescapeCharacters(text), proxyPort), nil
}
