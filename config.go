package sshc

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	wildcard "github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
)

const hostAny = "*"

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

type config struct {
	path    string
	content []byte
}

type configs []config

type identityKey struct {
	pattern    string
	key        []byte
	passphrase []byte
}

type identityFile struct {
	pattern    string
	path       string
	passphrase []byte
}

// Config is the type for the SSH Client config. not ssh_config.
type Config struct {
	configs         configs
	hostname        string
	user            string
	port            int
	identityFiles   []identityFile
	identityKeys    []identityKey
	passphrase      []byte
	useAgent        bool
	sshConfigs      []*sshConfig
	knownhosts      []string
	password        string
	auth            []ssh.AuthMethod
	dialTimeoutFunc func(network, addr string, timeout time.Duration) (net.Conn, error)
}

// Option is the type for change Config.
type Option func(*Config) error

// NewConfig creates SSH client config.
func NewConfig(options ...Option) (*Config, error) {
	var err error
	c := &Config{
		useAgent: true, // Default is true
	}
	base, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for _, p := range defaultConfigPaths {
		p, err := expandPath(p, base)
		if err != nil {
			return nil, err
		}
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		c.configs = appendConfig(c.configs, config{
			path:    p,
			content: b,
		})
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
	for _, cc := range c.configs {
		buf := new(bytes.Buffer)
		s := bufio.NewScanner(bytes.NewReader(cc.content))
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
		c.sshConfigs = append([]*sshConfig{{path: cc.path, sc: cfg}}, c.sshConfigs...)
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

func (c *Config) getKeyAndPassphrases(host string) ([]KeyAndPassphrase, error) {
	keys := []KeyAndPassphrase{}
	if len(c.identityKeys) > 0 {
		for _, i := range c.identityKeys {
			if wildcard.Match(i.pattern, host) {
				keys = append(keys, KeyAndPassphrase{
					key:        i.key,
					passphrase: i.passphrase,
				})
			}
		}
		for _, i := range c.identityFiles {
			if wildcard.Match(i.pattern, host) {
				b, err := os.ReadFile(i.path)
				if err != nil {
					return nil, err
				}
				keys = append(keys, KeyAndPassphrase{
					path:       i.path,
					key:        b,
					passphrase: i.passphrase,
				})
			}
		}
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
	if _, err := os.Lstat(keyPath); err != nil {
		return keys, nil
	}
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	keys = append(keys, KeyAndPassphrase{
		key:        b,
		passphrase: c.passphrase,
		path:       keyPath,
	})
	return keys, nil
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

// IdentityFile returns Option that append to Config.identityKeys for SSH client identity file.
func IdentityFile(f string, hostPatterns ...string) Option {
	return func(c *Config) error {
		opt := IdentityFileWithPassphrase(f, nil, hostPatterns...)
		return opt(c)
	}
}

// IdentityFileWithPassphrase returns Option that append to Config.identityKeys for SSH client identity file.
func IdentityFileWithPassphrase(f string, passphrase []byte, hostPatterns ...string) Option {
	return func(c *Config) error {
		if len(hostPatterns) == 0 {
			c.identityFiles = append(c.identityFiles, identityFile{
				pattern:    hostAny,
				path:       f,
				passphrase: passphrase,
			})
		} else {
			for _, pattern := range hostPatterns {
				c.identityFiles = append(c.identityFiles, identityFile{
					pattern:    pattern,
					path:       f,
					passphrase: passphrase,
				})
			}
		}
		return nil
	}
}

// IdentityKey returns Option that append to Config.identityKeys for SSH client identity file.
func IdentityKey(b []byte, hostPatterns ...string) Option {
	return func(c *Config) error {
		opt := IdentityKeyWithPassphrase(b, nil, hostPatterns...)
		return opt(c)
	}
}

// IdentityKeyWithPassphrase returns Option that append to Config.identityKeys for SSH client identity file.
func IdentityKeyWithPassphrase(b, passphrase []byte, hostPatterns ...string) Option {
	return func(c *Config) error {
		if len(hostPatterns) == 0 {
			c.identityKeys = append(c.identityKeys, identityKey{
				pattern:    hostAny,
				key:        b,
				passphrase: passphrase,
			})
		} else {
			for _, pattern := range hostPatterns {
				c.identityKeys = append(c.identityKeys, identityKey{
					pattern:    pattern,
					key:        b,
					passphrase: passphrase,
				})
			}
		}
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

// DialTimeoutFunc returns Option that set Config.dialTimeoutFunc for set SSH client dial func.
func DialTimeoutFunc(fn func(network, addr string, timeout time.Duration) (net.Conn, error)) Option {
	return func(c *Config) error {
		c.dialTimeoutFunc = fn
		return nil
	}
}

// ConfigData returns Option that unshift ssh_config data to Config.configs (alias of UnshiftConfigPath).
func ConfigData(b []byte) Option {
	return UnshiftConfigData(b)
}

// UnshiftConfigData returns Option that unshift ssh_config data to Config.configs.
func UnshiftConfigData(b []byte) Option {
	return func(c *Config) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r := sha256.Sum256(b)
		c.configs = unshiftConfig(c.configs, config{
			path:    filepath.Join(wd, string(r[:])),
			content: b,
		})
		return nil
	}
}

// AppendConfigData returns Option that append ssh_config data to Config.configs.
func AppendConfigData(b []byte) Option {
	return func(c *Config) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r := sha256.Sum256(b)
		c.configs = appendConfig(c.configs, config{
			path:    filepath.Join(wd, string(r[:])),
			content: b,
		})
		return nil
	}
}

// ConfigPath returns Option that unshift ssh_config path to Config.configs (alias of UnshiftConfigPath).
func ConfigPath(p string) Option {
	return UnshiftConfigPath(p)
}

// UnshiftConfigPath returns Option that unshift ssh_config path to Config.configs.
func UnshiftConfigPath(p string) Option {
	return func(c *Config) error {
		base, err := os.Getwd()
		if err != nil {
			return err
		}
		p, err := expandPath(p, base)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		c.configs = unshiftConfig(c.configs, config{
			path:    p,
			content: b,
		})
		return nil
	}
}

// AppendConfigPath returns Option that append ssh_config path to Config.configs.
func AppendConfigPath(p string) Option {
	return func(c *Config) error {
		base, err := os.Getwd()
		if err != nil {
			return err
		}
		p, err := expandPath(p, base)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		c.configs = appendConfig(c.configs, config{
			path:    p,
			content: b,
		})
		return nil
	}
}

// ClearConfig returns Option that clear Config.configs.
func ClearConfig() Option {
	return func(c *Config) error {
		c.configs = configs{}
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

// Password returns Option that override Config.password.
func Password(pass string) Option {
	return func(c *Config) error {
		c.password = pass
		return nil
	}
}

// AuthMethod returns Option that append ssh.AuthMethod to Config.auth.
func AuthMethod(m ssh.AuthMethod) Option {
	return func(c *Config) error {
		c.auth = append(c.auth, m)
		return nil
	}
}

func appendConfig(cs configs, c config) configs {
	return uniqueConfig(append(cs, c))
}

func uniqueConfig(cs configs) configs {
	keys := make(map[string]bool)
	l := configs{}
	for _, e := range cs {
		if _, v := keys[e.path]; !v {
			keys[e.path] = true
			l = append(l, e)
		}
	}
	return l
}

func unshiftConfig(cs configs, c config) configs {
	return uniqueConfig(append(configs{c}, cs...))
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
