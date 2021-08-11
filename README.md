[![Go Report Card](https://goreportcard.com/badge/github.com/gatherstars-com/jwz?style=flat-square)](https://goreportcard.com/report/github.com/github.com/gatherstars-com/jwz)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/github.com/gatherstars-com)](https://pkg.go.dev/github.com/github.com/gatherstars-com)
[![Release](https://img.shields.io/github/release/gatherstars-com/jwz.svg?style=flat-square)](https://github.com/gatherstars-com/jwz/releases/latest)
# The JWZ Threading algorithm written in Go

<img src="https://github.com/gatherstars-com/jwz/raw/master/docs/img/clonobg.png" alt="Gather Stars Logo" width="150" height="150" style="vertical-align: text-top; float: right; margin-left: 10px; margin-top: 0px"> 
This is an open source Go implementation of the widely known JWZ message threading algorithm originally written by 
Jamie Zawinsky - see https://www.jwz.org/doc/threading.html[his explanation here]. 

You will find an example of implementing the interface(s) needed to operate this package under the examples directory.

This module is intended to be production quality, but is currently a 0.9.x release because only the author has used it
to date. Do not hesitate to raise issues or enhancement requests or send pull requests. The 1.0.0 release may contain
breaking changes, but they will be trivial to adapt to. That release may also supply a few useful utility funcs.

More documentation will follow as we make this fully production ready.

```go
  include "github.com/gatherstars-com/jwz"
```

```shell
  ~/myproject: go mod tidy
```
