# sshc

SSH client using ~/.ssh/config

## Usage

Describe `~/.ssh/config` as follows

```
Host myhost
  HostName 203.0.113.1
  User k1low
  Port 10022
  IdentityFile ~/.ssh/myhost_rsa
```

`sshc.NewClient()` returns `*ssh.Client` using `~/.ssh/config`

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

## References

- https://github.com/kevinburke/ssh_config
