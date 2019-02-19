package sshc

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ScaleFT/sshkeys"
	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// Config return SSH Client config
type Config struct {
	host       string
	user       string
	port       int
	passphrase []byte
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

// NewClient return *ssh.Client using ~/.ssh/config
func NewClient(host string, options ...Option) (*ssh.Client, error) {
	port, err := strconv.Atoi(ssh_config.Get(host, "Port"))
	if err != nil {
		return nil, err
	}
	c := &Config{
		host:       host,
		user:       ssh_config.Get(host, "User"),
		port:       port,
		passphrase: []byte{},
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
	user := c.user
	port := strconv.Itoa(c.port)
	hostname, _ := c.getHostname()
	v = strings.Replace(v, "%h", hostname, -1)
	v = strings.Replace(v, "%p", port, -1)
	v = strings.Replace(v, "%r", user, -1)
	return v
}

func (c *Config) getHostname() (string, error) {
	h, err := ssh_config.GetStrict(c.host, "Hostname")
	if err != nil {
		return "", err
	}
	if h == "" {
		return c.host, nil
	}
	return h, nil
}

func (c *Config) getIdentityFile() (string, error) {
	keyPath, err := ssh_config.GetStrict(c.host, "IdentityFile")
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
