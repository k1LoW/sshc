package sshc

// Option is the type for change Config.
type Option func(*Config) error

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
