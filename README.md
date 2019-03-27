# sshc [![Build Status](https://travis-ci.org/k1LoW/sshc.svg?branch=master)](https://travis-ci.org/k1LoW/sshc) [![codecov](https://codecov.io/gh/k1LoW/sshc/branch/master/graph/badge.svg)](https://codecov.io/gh/k1LoW/sshc)

`sshc.NewClient()` returns `*ssh.Client` using [ssh_config(5)](https://linux.die.net/man/5/ssh_config)

## Usage

Describe `~/.ssh/config`.

```
Host myhost
  HostName 203.0.113.1
  User k1low
  Port 10022
  IdentityFile ~/.ssh/myhost_rsa
```

Use `sshc.NewClient()` as follows

``` go
package main

import (
	"bytes"
	"log"

	"github.com/k1LoW/sshc"
)

func main() {
	client, err := sshc.NewClient("myhost")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer session.Close()
	var stdout = &bytes.Buffer{}
	session.Stdout = stdout
	err = session.Run("hostname")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("result: %s", stdout.String())
}
```

## sshc.Option

``` go
client, err := sshc.NewClient("myhost", User("k1low"), Port(1022))
```

Available options

- User
- Port
- Passphrase
- ConfigPath ( Default is `~/.ssh/config` and `/etc/ssh/ssh_config` )
- UseAgent ( Default is `true` )

## References

- https://github.com/kevinburke/ssh_config
