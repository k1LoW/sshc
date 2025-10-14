# Changelog

## [v4.3.1](https://github.com/k1LoW/sshc/compare/v4.3.0...v4.3.1) - 2025-10-14
- test: add check license using go-licenses by @k1LoW in https://github.com/k1LoW/sshc/pull/59
- fix: retract versions by @k1LoW in https://github.com/k1LoW/sshc/pull/60

## [v4.3.0](https://github.com/k1LoW/sshc/compare/v4.2.1...v4.3.0) - 2025-10-08
- fix: use go-wildcard instead by @k1LoW in https://github.com/k1LoW/sshc/pull/54
- Bump golang.org/x/crypto from 0.17.0 to 0.35.0 in /example/ssh by @dependabot[bot] in https://github.com/k1LoW/sshc/pull/56

## [v4.2.1](https://github.com/k1LoW/sshc/compare/v4.2.0...v4.2.1) - 2025-07-25

## [v4.2.0](https://github.com/k1LoW/sshc/compare/v4.1.1...v4.2.0) - 2024-01-17
- Add DialTimeoutFunc option by @k1LoW in https://github.com/k1LoW/sshc/pull/50

## [v4.1.1](https://github.com/k1LoW/sshc/compare/v4.1.0...v4.1.1) - 2023-12-19
- Bump golang.org/x/crypto from 0.8.0 to 0.17.0 in /example/ssh by @dependabot in https://github.com/k1LoW/sshc/pull/47
- Bump golang.org/x/crypto from 0.8.0 to 0.17.0 by @dependabot in https://github.com/k1LoW/sshc/pull/49

## [v4.1.0](https://github.com/k1LoW/sshc/compare/v4.0.1...v4.1.0) - 2023-06-11
- Store key path in KeyAndPassphrase by @k1LoW in https://github.com/k1LoW/sshc/pull/45

## [v4.0.1](https://github.com/k1LoW/sshc/compare/v4.0.0...v4.0.1) - 2023-04-12
- Allow use of specified key without ~/.ssh/id_rsa by @k1LoW in https://github.com/k1LoW/sshc/pull/43

## [v4.0.0](https://github.com/k1LoW/sshc/compare/v3.2.0...v4.0.0) - 2023-04-10
- [BREAKING CHANGE] Error if non-existent config path is specified by @k1LoW in https://github.com/k1LoW/sshc/pull/38
- Update pkgs by @k1LoW in https://github.com/k1LoW/sshc/pull/40
- [BREAKING CHANGE] Support for multiple identity keys and passphrases by @k1LoW in https://github.com/k1LoW/sshc/pull/41
- [BREAKING CHANGE] Add options for adding ssh_config data by @k1LoW in https://github.com/k1LoW/sshc/pull/42

## [v3.2.0](https://github.com/k1LoW/sshc/compare/v3.1.0...v3.2.0) - 2023-02-19
- Support Windows ( use k1LoW/exec instead of direct syscall ) by @k1LoW in https://github.com/k1LoW/sshc/pull/35
- Update go and pkgs by @k1LoW in https://github.com/k1LoW/sshc/pull/37

## [v3.1.0](https://github.com/k1LoW/sshc/compare/v3.0.1...v3.1.0) - 2023-02-10
- Add `AuthMethod` option for appending ssh.AuthMethod by @k1LoW in https://github.com/k1LoW/sshc/pull/33

## [v3.0.1](https://github.com/k1LoW/sshc/compare/v3.0.0...v3.0.1) - 2022-11-28
- Fix `module` section of go.mod  by @k1LoW in https://github.com/k1LoW/sshc/pull/31

## [v3.0.0](https://github.com/k1LoW/sshc/compare/v2.4.1...v3.0.0) - 2022-11-28
- [BREAKING] Fix identity file path resolution / Fix proxyCommand execute working directory by @k1LoW in https://github.com/k1LoW/sshc/pull/26
- Update README by @k1LoW in https://github.com/k1LoW/sshc/pull/30

## [v2.4.1](https://github.com/k1LoW/sshc/compare/v2.4.0...v2.4.1) - 2022-11-28
- Use tagpr by @k1LoW in https://github.com/k1LoW/sshc/pull/27

## [v2.4.0](https://github.com/k1LoW/sshc/compare/v2.3.0...v2.4.0) (2022-11-28)

* Fix Config.Get() [#25](https://github.com/k1LoW/sshc/pull/25) ([k1LoW](https://github.com/k1LoW))
* Add `IdentityFile` and `IdentityKey` option [#24](https://github.com/k1LoW/sshc/pull/24) ([k1LoW](https://github.com/k1LoW))
* Support hostname option [#23](https://github.com/k1LoW/sshc/pull/23) ([k1LoW](https://github.com/k1LoW))

## [v2.3.0](https://github.com/k1LoW/sshc/compare/v2.2.0...v2.3.0) (2022-11-26)

* Bump up go and pkgs version [#22](https://github.com/k1LoW/sshc/pull/22) ([k1LoW](https://github.com/k1LoW))
* Use panubo/sshd:latest instead of k1low/sshd [#21](https://github.com/k1LoW/sshc/pull/21) ([k1LoW](https://github.com/k1LoW))
* Use octocov [#20](https://github.com/k1LoW/sshc/pull/20) ([k1LoW](https://github.com/k1LoW))

## [v2.2.0](https://github.com/k1LoW/sshc/compare/v2.1.1...v2.2.0) (2021-09-27)

* Support timeout [#19](https://github.com/k1LoW/sshc/pull/19) ([k1LoW](https://github.com/k1LoW))

## [v2.1.1](https://github.com/k1LoW/sshc/compare/v2.1.0...v2.1.1) (2021-08-23)


## [v2.1.0](https://github.com/k1LoW/sshc/compare/v2.0.0...v2.1.0) (2021-08-23)

* Support password [#18](https://github.com/k1LoW/sshc/pull/18) ([k1LoW](https://github.com/k1LoW))

## [v2.0.0](https://github.com/k1LoW/sshc/compare/v1.3.0...v2.0.0) (2021-08-23)

* v2 [#17](https://github.com/k1LoW/sshc/pull/17) ([k1LoW](https://github.com/k1LoW))

## [v1.3.0](https://github.com/k1LoW/sshc/compare/v1.2.0...v1.3.0) (2021-08-22)

* Support Include keyword of root [#16](https://github.com/k1LoW/sshc/pull/16) ([k1LoW](https://github.com/k1LoW))
