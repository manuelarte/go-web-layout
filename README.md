# Go Web Layout

[![CI](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml/badge.svg)](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/manuelarte/go-web-layout)](https://goreportcard.com/report/github.com/manuelarte/go-web-layout)

A production-ready template for Go web applications featuring modern tooling and best practices.

## 🚀 Key Features

### Database Layer

#### Migrations

Managed via [go-embed](https://pkg.go.dev/embed) migration files in [./resources/migrations](./resources/migrations) using [golang-migrate](https://github.com/golang-migrate/migrate).

#### Type-safe SQL

All queries generated at compile time with [sqlc](https://sqlc.dev/).

### API Layers

#### REST API

API specification in [openapi.yml](openapi.yml) and code automatically generated with [oapi-codegen](https://github.com/deepmap/oapi-codegen).

#### gRPC API

The gRPC API is defined in the folder [proto/](proto), and the generated code with [buf](https://buf.build/) is in the folder [./internal/api/grpc](./internal/api/grpc).

## 🛠️ Getting Started

Run the project by:

```bash
go run ./cmd/go-web-layout/.
```

Or you can run it with docker using:

```bash
make dr
```
