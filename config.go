package sshc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

var (
	defaultConfigPaths = []string{
		filepath.Join("~", ".ssh", "config"),
		filepath.Join("/", "etc", "ssh", "ssh_config"),
	}
	includeRelRe  = regexp.MustCompile(`^(Include\s+~)(.+)$`)
	includeRelRe2 = regexp.MustCompile(`^(Include\s+)([^~/].+)$`)
)

// Config is the type for the SSH Client config. not ssh_config.
type Config struct {
	configPaths []string
	user        string
	port        int
	passphrase  []byte
	useAgent    bool
	configs     []*ssh_config.Config
	knownhosts  []string
	password    string
}

// Option is the type for change Config.
type Option func(*Config) error

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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	for _, p := range c.configPaths {
		cPath := strings.Replace(p, "~", homeDir, 1)
		if _, err := os.Lstat(cPath); err != nil {
			continue
		}
		f, err := os.Open(filepath.Clean(cPath))
		if err != nil {
			return nil, err
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
				return nil, err
			}
		}

		cfg, err := ssh_config.Decode(buf)
		if err != nil {
			return nil, err
		}
		c.configs = append([]*ssh_config.Config{cfg}, c.configs...)
	}

	return c, nil
}

// Get returns Config value.
func (c *Config) Get(host, key string) string {
	// Return the value overridden by option
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

// User returns Option that set Config.user for override SSH client user.
func User(u string) Option {
	return func(c *Config) error {
		c.user = u
		return nil
	}
}

// Port returns Option that set Config.port for override SSH client port.
func Port(p int) Option {
	return func(c *Config) error {
		c.port = p
		return nil
	}
}

// Passphrase returns Option that set Config.passphrase for set SSH key passphrase.
func Passphrase(p []byte) Option {
	return func(c *Config) error {
		c.passphrase = p
		return nil
	}
}

// ConfigPath is alias of UnshiftConfigPath.
func ConfigPath(p string) Option {
	return UnshiftConfigPath(p)
}

// UnshiftConfigPath returns Option that unshift ssh_config path to Config.configpaths.
func UnshiftConfigPath(p string) Option {
	return func(c *Config) error {
		c.configPaths = unique(append([]string{p}, c.configPaths...))
		return nil
	}
}

// AppendConfigPath returns Option that append ssh_config path to Config.configpaths.
func AppendConfigPath(p string) Option {
	return func(c *Config) error {
		c.configPaths = unique(append(c.configPaths, p))
		return nil
	}
}

// ClearConfigPath returns Option thet clear Config.configpaths,
func ClearConfigPath() Option {
	return func(c *Config) error {
		c.configPaths = []string{}
		return nil
	}
}

// UseAgent returns Option that override Config.useAgent.
func UseAgent(u bool) Option {
	return func(c *Config) error {
		c.useAgent = u
		return nil
	}
}

// Knownhosts returns Option that override Config.knownhosts.
func Knownhosts(files ...string) Option {
	return func(c *Config) error {
		c.knownhosts = files
		return nil
	}
}

// Password returns Option that override Config.password
func Password(pass string) Option {
	return func(c *Config) error {
		c.password = pass
		return nil
	}
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