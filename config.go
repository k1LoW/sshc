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
	"golang.org/x/crypto/ssh"
)

var (
	defaultConfigPaths = []string{
		filepath.Join("~", ".ssh", "config"),
		filepath.Join("/", "etc", "ssh", "ssh_config"),
	}
	includeRelRe  = regexp.MustCompile(`^(Include\s+~)(.+)$`)
	includeRelRe2 = regexp.MustCompile(`^(Include\s+)([^~/].+)$`)
)

type sshConfig struct {
	sc   *ssh_config.Config
	path string
}

// Config is the type for the SSH Client config. not ssh_config.
type Config struct {
	configPaths []string
	hostname    string
	user        string
	port        int
	identityKey []byte
	passphrase  []byte
	useAgent    bool
	sshConfigs  []*sshConfig
	knownhosts  []string
	password    string
	auth        []ssh.AuthMethod
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
		cPath := filepath.Clean(strings.Replace(p, "~", homeDir, 1))
		if _, err := os.Lstat(cPath); err != nil {
			continue
		}
		f, err := os.Open(cPath)
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := s.Bytes()

			// Replace include path
			if includeRelRe.Match(line) {
				line = includeRelRe.ReplaceAll(line, []byte(fmt.Sprintf("Include %s$2", homeDir)))
			} else if includeRelRe2.Match(line) {
				line = includeRelRe2.ReplaceAll(line, []byte(fmt.Sprintf("Include %s/.ssh/$2", homeDir)))
			}

			if _, err := buf.Write(append(line, []byte("\n")...)); err != nil {
				return nil, err
			}
		}

		cfg, err := ssh_config.Decode(buf)
		if err != nil {
			return nil, err
		}
		c.sshConfigs = append([]*sshConfig{{path: cPath, sc: cfg}}, c.sshConfigs...)
	}

	return c, nil
}

// Get returns Config value.
func (c *Config) Get(host, key string) string {
	// Return the value overridden by option
	switch key {
	case "User":
		return c.getUser(host)
	case "Port":
		p, err := c.getPort(host)
		if err != nil {
			return ""
		}
		return strconv.Itoa(p)
	case "Hostname":
		h, err := c.getHostname(host)
		if err != nil {
			return ""
		}
		return h
	}
	return c.getRaw(host, key)
}

func (c *Config) getRaw(host, key string) string {
	for _, scs := range c.sshConfigs {
		val, err := scs.sc.Get(host, key)
		if err != nil || val != "" {
			return val
		}
	}
	return ssh_config.Default(key)
}

func (c *Config) getRawWithBase(host, key string) (string, string) {
	for _, scs := range c.sshConfigs {
		val, err := scs.sc.Get(host, key)
		if err != nil || val != "" {
			return val, filepath.Dir(scs.path)
		}
	}
	return ssh_config.Default(key), ""
}

func (c *Config) getUser(host string) string {
	if c.user != "" {
		return c.user
	}
	return c.getRaw(host, "User")
}

func (c *Config) getPort(host string) (int, error) {
	if c.port != 0 {
		return c.port, nil
	}
	p := c.getRaw(host, "Port")
	return strconv.Atoi(p)
}

func (c *Config) getHostname(host string) (string, error) {
	if c.hostname != "" {
		return c.hostname, nil
	}
	h := c.getRaw(host, "Hostname")
	if h == "" {
		return host, nil
	}
	return h, nil
}

func (c *Config) getIdentityKey(host string) ([]byte, error) {
	if c.identityKey != nil {
		return c.identityKey, nil
	}

	user := c.getUser(host)
	port, err := c.getPort(host)
	if err != nil {
		return nil, err
	}
	hostname, err := c.getHostname(host)
	if err != nil {
		return nil, err
	}

	keyPath, base := c.getRawWithBase(host, "IdentityFile")
	keyPath = expandVerbs(keyPath, user, port, hostname)
	keyPath, err = expandPath(keyPath, base)
	if err != nil {
		return nil, err
	}
	if i, _ := expandPath("~/.ssh/identity", base); keyPath == i {
		if _, err := os.Lstat(i); err != nil {
			keyPath, err = expandPath("~/.ssh/id_rsa", base)
			if err != nil {
				return nil, err
			}
		}
	}

	return os.ReadFile(keyPath)
}

func (c *Config) getProxyCommand(host string) (string, string) {
	return c.getRawWithBase(host, "ProxyCommand")
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

// Hostname returns Option that set Config.hostname for override SSH client port.
func Hostname(h string) Option {
	return func(c *Config) error {
		c.hostname = h
		return nil
	}
}

// IdentityFile returns Option that set Config.identityKey for override SSH client identity file.
func IdentityFile(p string) Option {
	return func(c *Config) error {
		key, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			return err
		}
		c.identityKey = key
		return nil
	}
}

// IdentityKey returns Option that set Config.identityKey for override SSH client identity file.
func IdentityKey(b []byte) Option {
	return func(c *Config) error {
		c.identityKey = b
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

// AuthMethod returns Option that append ssh.AuthMethod to Config.auth
func AuthMethod(m ssh.AuthMethod) Option {
	return func(c *Config) error {
		c.auth = append(c.auth, m)
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

func expandPath(path, base string) (string, error) {
	switch {
	case strings.HasPrefix(path, "/"):
		return path, nil
	case strings.HasPrefix(path, "~"):
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Clean(strings.Replace(path, "~", homeDir, 1)), nil
	default:
		return filepath.Clean(filepath.Join(base, path)), nil
	}
}
