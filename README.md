# Confita

[![Build Status](https://travis-ci.org/heetch/confita.svg?branch=master)](https://travis-ci.org/heetch/confita)
[![GoDoc](https://godoc.org/github.com/heetch/confita?status.svg)](https://godoc.org/github.com/heetch/confita)
[![Go Report Card](https://goreportcard.com/badge/github.com/heetch/confita)](https://goreportcard.com/report/github.com/heetch/confita)

Confita is a library that loads configuration from multiple backends and stores it in a struct.

## Install

```sh
go get -u github.com/heetch/confita
```

## Usage

Confita scans a struct for `config` tags and calls all the backends one after another (except if the [backend tag](#backend-tag) is specified) until the key is found.
The value is then converted into the type of the field.

### Struct layout

Go primitives are supported:

```go
type Config struct {
  Host        string        `config:"host"`
  Port        uint32        `config:"port"`
  Timeout     time.Duration `config:"timeout"`
}
```

By default, all fields are optional. With the required option, if a key is not found then Confita will return an error.

```go
type Config struct {
  Addr        string        `config:"addr,required"`
  Timeout     time.Duration `config:"timeout"`
}
```

Nested structs are supported too:

```go
type Config struct {
  Host        string        `config:"host"`
  Port        uint32        `config:"port"`
  Timeout time.Duration     `config:"timeout"`
  Database struct {
    URI string              `config:"database-uri,required"`
  }
```

As a special case, if the field tag is "-", the field is always omitted. This is useful if you want to populate this field on your own.

```go
type Config struct {
  // Field is ignored by this package.
  Field float64 `config:"-"`

  // Confita scans any structure recursively, the "-" value prevents that.
  Client http.Client `config:"-"`
}
```

### Backend tag

As we already said, Confita will call each backend one after another. But in order to avoid some useless processing we can specify in which backend the value of a field can be found with the `backend` tag:

```go
type Config struct {
  Host        string        `config:"host" backend:"env"`
  Port        uint32        `config:"port" backend:"etcd"`
  Timeout     time.Duration `config:"timeout"`
}
```

Thanks to this tag, the engine will search for those values exclusively in the specified backends.

### Loading configuration

Creating a loader:

```go
loader := confita.NewLoader()
```

By default, a Confita loader loads all the keys from the environment.
A loader can take other configured backends as parameters.

#### Supported backends:

- Environment variables
- JSON files
- Yaml files
- [etcd](https://github.com/coreos/etcd)
- [Consul](https://www.consul.io/)

```go
loader := confita.NewLoader(
  env.NewBackend(),
  file.NewBackend("/path/to/config.json"),
  file.NewBackend("/path/to/config.yaml"),
  etcd.NewBackend(etcdClientv3, "namespace"),
  consul.NewBackend(consulClient, "namespace"),
)
```

Loading configuration:

```go
err := loader.Load(context.Background(), &cfg)
```

Since loading configuration can take time when used with multiple remote backends, context can be used for timeout and cancelation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
defer cancel()
err := loader.Load(ctx, &cfg)
```

### Default values

If a key is not found, Confita won't change the respective struct field. With that in mind, default values can simply be implemented by filling the structure before passing it to Confita.

```go

type Config struct {
  Host        string        `config:"host"`
  Port        uint32        `config:"port"`
  Timeout     time.Duration `config:"timeout"`
  Password    string        `config:"password,required"`
}

// default values
cfg := Config{
  Host: "127.0.0.1",
  Port: "5656",
  Timeout: 5 * time.Second,
}

err := confita.NewLoader().Load(context.Background(), &cfg)
```

## License

The library is released under the MIT license. See LICENSE file.
