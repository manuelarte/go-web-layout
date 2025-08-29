# Go web layout

[![CI](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml/badge.svg)](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/manuelarte/go-web-layout)](https://goreportcard.com/report/github.com/manuelarte/go-web-layout)

A production-ready template for Go web applications featuring modern tooling and best practices.

## 🚀 Key features

### 🛢 Database layer

- **Migrations**:

The library [golang-migrate](https://github.com/golang-migrate/migrate) handles the database migrations.
Go's feature [go-embed](https://pkg.go.dev/embed) allows to embed the migrations files in the binary.

- **Type-safe SQL**:

The library [sqlc](https://sqlc.dev/) generates all type-safe queries at compile time.

### 🌐 Application programming interface layers

- **REST API**:

API specification in [openapi.yml](resources/openapi.yml) and code automatically generated with [oapi-codegen](https://github.com/deepmap/oapi-codegen).

> [!NOTE]
> Swagger UI endpoint available at [/swagger/index.html](http://localhost:3001/swagger/index.html).
>
> Prometheus metrics endpoint available at [/metrics](http://localhost:3001/metrics)

- **gRPC API**:

The folder [proto/](proto) contains the definition of the [gRPC](https://grpc.io/) API.
The library [buf](https://buf.build/) generates the code from that interface definition in the folder [./internal/api/grpc](./internal/api/grpc).

## Linters

The following linters keep the standard/best practices and consistency of the project:

### Golangci-lint

[Golangci-lint](https://golangci-lint.run/) is the most common linter for Go.
The file [.golangci.yml](.golangci.yml) contains its configuration.

### Buf

The library [buf](https://buf.build/) is a linter for [Protocol Buffers](https://developers.google.com/protocol-buffers).
The file [buf.gen.yaml](buf.gen.yaml) contains its configuration.

### Spectral

[Spectral](https://stoplight.io/open-source/spectral) is a linter for [OpenAPI](https://swagger.io/specification/) and [AsyncAPI](https://www.asyncapi.com/).
The file [.spectral.yaml](.spectral.yaml) contains its configuration.

## 🛠️ Getting started

Run the project by:

```bash
go run ./cmd/go-web-layout/.
```

Or you can run it with docker using:

```bash
make dr
```
