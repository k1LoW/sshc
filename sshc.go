// Package sshc provides sshc.NewClient() that returns *ssh.Client using ssh_config(5)
package sshc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ScaleFT/sshkeys"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
)

var defaultConfigPaths = []string{
	filepath.Join("~", ".ssh", "config"),
	filepath.Join("/", "etc", "ssh", "ssh_config"),
}

var includeRelRe = regexp.MustCompile(`^(Include\s+~)(.+)$`)
var includeRelRe2 = regexp.MustCompile(`^(Include\s+)([^~/].+)$`)

// Config is the type for the SSH Client config. not ssh_config.
type Config struct {
	configPaths []string
	user        string
	port        int
	passphrase  []byte
	useAgent    bool
	configs     []*ssh_config.Config
	loader      sync.Once
	knownhosts  []string
}

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
}

// NewConfig creates SSH client config.
func NewConfig(options ...Option) (*Config, error) {
	var err error
	c := &Config{
		configPaths: defaultConfigPaths,
		useAgent:    true, // Default is true
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
	c, err := NewConfig(options...)
	if err != nil {
		return nil, err
	}
	dc := &DialConfig{
		User: c.Get(host, "User"),
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
	identityFile, err := c.getIdentityFile(host)
	if err != nil {
		return nil, err
	}
	dc.IdentityFile = identityFile
	dc.ProxyCommand = c.Get(host, "ProxyCommand")
	dc.ProxyJump = c.Get(host, "ProxyJump")

	return Dial(dc)
}

// Get returns Config value.
func (c *Config) Get(host, key string) string {
	homeDir, err := os.UserHomeDir()
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

			buf := new(bytes.Buffer)
			s := bufio.NewScanner(f)
			for s.Scan() {
				line := s.Bytes()

				// Replace include path
				if includeRelRe.Match(line) {
					line = includeRelRe.ReplaceAll(line, []byte(fmt.Sprintf("Include %s$2", os.Getenv("HOME"))))
				} else if includeRelRe2.Match(line) {
					line = includeRelRe2.ReplaceAll(line, []byte(fmt.Sprintf("Include %s/.ssh/$2", os.Getenv("HOME"))))
				}

				if _, err := buf.Write(append(line, []byte("\n")...)); err != nil {
					panic(err)
				}
			}

			cfg, err := ssh_config.Decode(buf)
			if err != nil {
				continue
			}
			c.configs = append([]*ssh_config.Config{cfg}, c.configs...)
		}
	})

	// option setting
	switch {
	case key == "User" && c.user != "":
		return c.user
	case key == "Port" && c.port != 0:
		return strconv.Itoa(c.port)
	}

	for _, cfg := range c.configs {
		val, err := cfg.Get(host, key)
		if err != nil || val != "" {
			return val
		}
	}
	return ssh_config.Default(key)
}

// Dial returns *ssh.Client using Config
func Dial(dc *DialConfig) (*ssh.Client, error) {
	user := dc.User
	port := dc.Port
	hostname := dc.Hostname
	keyPath := dc.IdentityFile
	addr := fmt.Sprintf("%s:%d", hostname, port)

	auth := []ssh.AuthMethod{}
	key, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return nil, err
	}
	signer, err := sshkeys.ParseEncryptedPrivateKey(key, dc.Passphrase)
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
		} else {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	} else {
		auth = append(auth, ssh.PublicKeys(signer))
	}

	cb, err := hostKeyCallback(dc.Knownhosts)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: cb,
	}

	proxyCommand := dc.ProxyCommand
	proxyJump := dc.ProxyJump

	if proxyCommand == "" && proxyJump != "" {
		parsedProxyJump, err := parseProxyJump(proxyJump)
		if err != nil {
			return nil, err
		}
		proxyCommand = unescapeCharacters(parsedProxyJump, user, strconv.Itoa(port), hostname)
	}

	if proxyCommand != "" {
		client, server := net.Pipe()
		proxyCommand = unescapeCharacters(proxyCommand, user, strconv.Itoa(port), hostname)
		cmd := exec.Command("sh", "-c", proxyCommand) // #nosec
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = server
		cmd.Stdout = server
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("proxy command:%s error:%s", proxyCommand, err)
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

func (c *Config) getHostname(host string) (string, error) {
	h := c.Get(host, "Hostname")
	if h == "" {
		return host, nil
	}
	return h, nil
}

func (c *Config) getIdentityFile(host string) (string, error) {
	user := c.Get(host, "User")
	port := c.Get(host, "Port")
	hostname, err := c.getHostname(host)
	if err != nil {
		return "", err
	}
	keyPath := c.Get(host, "IdentityFile")
	keyPath = unescapeCharacters(keyPath, user, port, hostname)
	homeDir, err := os.UserHomeDir()
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
