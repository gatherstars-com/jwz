[![Go Report Card](https://goreportcard.com/badge/github.com/gatherstars-com/jwz?style=flat-square)](https://goreportcard.com/report/github.com/gatherstars-com/jwz)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/github.com/gatherstars-com/jwz)](https://pkg.go.dev/github.com/gatherstars-com/jwz)
[![Release](https://img.shields.io/github/v/release/gatherstars-com/jwz?sort=semver&style=flat-square)](https://github.com/gatherstars-com/jwz/releases/latest)
[![Release](https://img.shields.io/github/go-mod/go-version/gatherstars-com/jwz?style=flat-square)](https://github.com/gatherstars-com/jwz/releases/latest)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg?style=flat-square)](https://github.com/gatherstars-com/jwz/commit-activity)
[![GitHub license](https://img.shields.io/github/license/gatherstars-com/jwz.svg)](https://www.gathersatrs.com)
[![GitHub stars](https://img.shields.io/github/stars/gatherstars-com/jwz.svg?style=flat-square&label=Star&maxAge=2592000)](https://GitHub.com/Naereen/StrapDown.js/stargazers/)
# The JWZ Threading algorithm written in Go

This is an open source Go implementation of the widely known JWZ message threading algorithm originally written by 
Jamie Zawinsky. See [his explanation here](https://www.jwz.org/doc/threading.html) 

You will find an example of implementing the interface(s) needed to operate this package in the `/examples/visualize` 
directory.

See the godoc for examples in documentation form, and the example in `examples\visualize`

## Functionality

The package provides the original JWZ algorithm to implementors of the `Threadable` interface. It has been tested against
many thousands of emails. The interface provides a few extra features over the original Java version, but these are
additions and enhancements to the interface, not algorithmic changes.

As well as providing the threading capability itself, the package also provides:

  - A generic walker, to which you can provide a function to operate upon the nodes in the threaded tree. 
  - A generic sorter, to which you can provide your own comparison function (a byDate example is provided)

Feel free to report any issues and offer any additions by pull requests.

## Quick start

```go
  include "github.com/gatherstars-com/jwz"
```

```shell
  ~/myproject $ go mod tidy
```

```go
	// Use the enmime package to build all the emails we find under the given directory and store them in a slice
	// of structs which implement the Threadable interface
	//
	//
	threadables, err := buildEnvelopes(testData, sizeHint)
	if err != nil {
		log.Printf("Unable to walk the eml directory: %#v", err)
		os.Exit(1)
	}

	// Now we have a big slice of all the emails, lets use our jwz algorithm to place them in to a thread tree
	//
	threader := jwz.NewThreader()
	sliceRoot, err := threader.ThreadSlice(threadables)
	if err != nil {
		log.Fatalf("Email threading operation return fatal error: %#v", err)
	}

	// Sort it Rodney!
	//
	x := jwz.Sort(sliceRoot, byDate)
```

An implementation of buildEnvelopes can be found in `examples\visualizer\handlers.go`