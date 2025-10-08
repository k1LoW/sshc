> [!CAUTION]
> **Correction of Licensing Error and Request for Action**<br>
> **Please upgrade version to v4.3.0 or later**<br>
> For details, see https://github.com/k1LoW/sshc/issues/57

# sshc [![Build Status](https://github.com/k1LoW/sshc/workflows/build/badge.svg)](https://github.com/k1LoW/sshc/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/k1LoW/sshc/v4.svg)](https://pkg.go.dev/github.com/k1LoW/sshc/v4) ![Coverage](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/sshc/coverage.svg) ![Code to Test Ratio](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/sshc/ratio.svg)

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

	"github.com/k1LoW/sshc/v4"
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

### sshc.Option

``` go
client, err := sshc.NewClient("myhost", User("k1low"), Port(1022))
```

See [godoc page](https://pkg.go.dev/github.com/k1LoW/sshc/v4#Option)

## Supported ssh_config keywords

- Hostname
- Port
- User
- IdentityFile
- ProxyCommand
- ProxyJump

## References

- https://github.com/kevinburke/ssh_config
