package sshc

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
)

// Config return SSH Client config
type Config struct {
	Host string
	User string
	Port int
}

// Option function change Config
type Option func(*Config) error

// User return Option set Config.User
func User(u string) Option {
	return func(c *Config) error {
		c.User = u
		return nil
	}
}

// Port return Option set Config.Port
func Port(p int) Option {
	return func(c *Config) error {
		c.Port = p
		return nil
	}
}

// NewClient return *ssh.Client using ~/.ssh/config
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	port, err := strconv.Atoi(ssh_config.Get(host, "Port"))
	if err != nil {
		return nil, err
	}
	c := &Config{
		Host: host,
		User: ssh_config.Get(host, "User"),
		Port: port,
	}
	for _, option := range options {
		err = option(c)
		if err != nil {
			return nil, err
		}
	}
	return c.DialWithConfig()
}

// DialWithConfig return *ssh.Client using ~/.ssh/config
func (c *Config) DialWithConfig() (*ssh.Client, error) {
	host := c.Host
	user := c.User
	port := strconv.Itoa(c.Port)
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
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	auth = append(auth, ssh.PublicKeys(signer))

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // FIXME
	}

	proxyCommand := ssh_config.Get(host, "ProxyCommand")
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
	user := c.User
	port := strconv.Itoa(c.Port)
	hostname, _ := c.getHostname()
	v = strings.Replace(v, "%h", hostname, -1)
	v = strings.Replace(v, "%p", port, -1)
	v = strings.Replace(v, "%r", user, -1)
	return v
}

func (c *Config) getHostname() (string, error) {
	h, err := ssh_config.GetStrict(c.Host, "Hostname")
	if err != nil {
		return "", err
	}
	if h == "" {
		return c.Host, nil
	}
	return h, nil
}

func (c *Config) getIdentityFile() (string, error) {
	keyPath, err := ssh_config.GetStrict(c.Host, "IdentityFile")
	if err != nil {
		return "", err
	}
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
