# Flagstruct

[![GoDoc](https://godoc.org/github.com/mfuentesg/flagstruct?status.svg)](https://godoc.org/github.com/mfuentesg/flagstruct)
[![Build Status](https://travis-ci.org/mfuentesg/flagstruct.svg?branch=master)](https://travis-ci.org/mfuentesg/flagstruct)
[![codecov](https://codecov.io/gh/mfuentesg/flagstruct/branch/master/graph/badge.svg)](https://codecov.io/gh/mfuentesg/flagstruct)

This library is inspired on [joeshaw/envdecode](https://github.com/joeshaw/envdecode) and [this video](https://youtu.be/PTE4VJIdHPg?t=7m50s).
Instead of read env variables, `flagstruct` help you to populate your structs from command line arguments.
`flagstruct` works with plain and nested structs, including pointers to nested structs. But, it will not allocate new points to structs.

**Considerations**

1. Default values may be provided by appending ",default=value" to the struct tag
2. Required values may be marked by appending ",required" to the struct tag
3. Allowed values may be provided by appending ",allowed=option;option..." to the struct tag 
4. `flagstruct` will ignore every unexported struct field (including one that contains no `flag` tags at all)
5. You can't use `default` and `required` in the same annotation

## Getting started

### Installation

```bash
$ go get github.com/mfuentesg/flagstruct
```

### Using it

Define a struct with `flag` annotation

```go
// main.go file
package main

import (
	"fmt"
	"time"

	"github.com/mfuentesg/flagstruct"
)

type Config struct {
	Server struct {
		Host     string `flag:"server-host,default=localhost"`
		Port     int    `flag:"server-port,allowed=9090;8080"`
	}
	Timeout time.Duration `flag:"timeout,default=1m"`
}
```
Then, call to `flagstruct.Decode` function
```go
func main() {
	var c Config
	if err := flagstruct.Decode(&c); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(
		"connecting to %s:%d (timeout %.0f seconds)",
		c.Server.Host,
		c.Server.Port,
		c.Timeout.Seconds(),
	)
}
```

**Running the above example**
```sh
$ go run main.go
> connecting to localhost:0 (timeout 60 seconds)

$ go run main.go -timeout=2m -server-port=3000
> connecting to localhost:3000 (timeout 120 seconds)
```

## Supported types

* Structs
* Pointer to structs
* Slices of below defined types, separated by semicolon (`;`)
* `bool`
* `float32`, `float64`
* `int`, `int8`, `int16`, `int32`, `int64`
* `uint`, `uint8`, `uint16`, `uint32`, `uint64`
* `string`
* `interface{}`
* `time.Duration`, using the [`time.ParseDuration()` format](http://golang.org/pkg/time/#ParseDuration)
* Custom types (those types must implement the `flagstruct.Decoder` interface)

## Custom `Decoder`

if you want that a field use a custom decoder, you may implement the `Decoder` interface.
> `Decoder` is the interface implemented by an object, that can decode an argument string representation of itself.

```go
type Config struct {
  IPAddr IP `flag:"ip"`
}

type IP net.IP

// Decode implements the interface `flagstruct.Decoder`
func (i *IP) Decode(repl string) error {
  *i = net.ParseIP(repl)
  return nil
}
```

