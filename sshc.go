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

	"golang.org/x/crypto/ssh/agent"

	"github.com/ScaleFT/sshkeys"
	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var defaultConfigPaths = []string{
	filepath.Join("~", ".ssh", "config"),
	filepath.Join("/", "etc", "ssh", "ssh_config"),
}

// Config return SSH Client config. not ssh_config
type Config struct {
	configPaths []string
	host        string
	user        string
	port        int
	passphrase  []byte
	configs     []*ssh_config.Config
	loader      sync.Once
}

// Option function change Config
type Option func(*Config) error

// User return Option set Config.user
func User(u string) Option {
	return func(c *Config) error {
		c.user = u
		return nil
	}
}

// Port return Option set Config.port
func Port(p int) Option {
	return func(c *Config) error {
		c.port = p
		return nil
	}
}

// Passphrase return Option set Config.passphrase
func Passphrase(p []byte) Option {
	return func(c *Config) error {
		c.passphrase = p
		return nil
	}
}

// ConfigPath is alias of UnshiftConfigPath
func ConfigPath(p string) Option {
	return UnshiftConfigPath(p)
}

// UnshiftConfigPath return Option unshift ssh_config path to Config.configpaths
func UnshiftConfigPath(p string) Option {
	return func(c *Config) error {
		c.configPaths = unique(append([]string{p}, c.configPaths...))
		return nil
	}
}

// AppendConfigPath return Option append ssh_config path to Config.configpaths
func AppendConfigPath(p string) Option {
	return func(c *Config) error {
		c.configPaths = unique(append(c.configPaths, p))
		return nil
	}
}

// ClearConfigPath return Option clear Config.configpaths
func ClearConfigPath() Option {
	return func(c *Config) error {
		c.configPaths = []string{}
		return nil
	}
}

// NewClient return *Config
func NewConfig(host string, options ...Option) (*Config, error) {
	var err error

	c := &Config{
		configPaths: defaultConfigPaths,
		host:        host,
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

// NewClient return *ssh.Client
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	c, err := NewConfig(host, options...)
	if err != nil {
		return nil, err
	}
	return c.DialWithConfig()
}

// Get return value
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

// DialWithConfig return *ssh.Client
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

	if sshAuthSockExists() {
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
