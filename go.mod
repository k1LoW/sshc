module github.com/k1LoW/sshc/v4

go 1.24.0

require (
	github.com/IGLOU-EU/go-wildcard/v2 v2.1.0
	github.com/ScaleFT/sshkeys v1.4.0
	github.com/k1LoW/exec v0.3.0
	github.com/kevinburke/ssh_config v1.2.0
	golang.org/x/crypto v0.45.0
	golang.org/x/term v0.37.0
)

require (
	github.com/dchest/bcrypt_pbkdf v0.0.0-20150205184540-83f37f9c154a // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

// Licensing error. ref: https://github.com/k1LoW/sshc/issues/57
retract [v4.0.0, v4.2.1]
