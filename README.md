# Sentry client for zap logger

Zap Core for Sentry

![CI Status](https://github.com/gebv/zapsentry/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/gebv/zapsentry)](https://goreportcard.com/report/github.com/gebv/zapsentry)

## Features

* with stacktrace (this is handy for analysis into Sentry UI)
* with tests (can always be tests)
* easy setup

# Quick start

Using [go mod](https://github.com/golang/go/wiki/Modules):

```
go get github.com/gebv/zapsentry@v2.0.0
```

Simple example:

```go
// TODO
```

# Version Policy

`zapsentry` follows semantic versioning for the documented public API on stable releases. `v2` is the latest stable version and follows [SemVer](http://semver.org/) strictly. Follows [changelog](./CHANGELOG.md).

The library `v1` after improved and finalized went into `v2`. `v2` has a module name `github.com/gebv/zapsentry/v2`
`v1` follows [TheZeroSlave/zapsentry](https://github.com/TheZeroSlave/zapsentry).

# License

MIT, see [LICENSE](./LICENSE).
