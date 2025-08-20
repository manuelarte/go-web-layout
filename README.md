# Go Web Layout

[![CI](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml/badge.svg)](https://github.com/manuelarte/go-web-layout/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/manuelarte/go-web-layout)](https://goreportcard.com/report/github.com/manuelarte/go-web-layout)

A production-ready template for Go web applications featuring modern tooling and best practices.

## 🚀 Key Features

### 🛢 Database Layer

- **Migrations**:

The database migration is handled with [golang-migrate](https://github.com/golang-migrate/migrate). The migration files are embedded using [go-embed](https://pkg.go.dev/embed).

- **Type-safe SQL**:

All queries are generated at compile time with [sqlc](https://sqlc.dev/).

### 🌐 API Layers

- **REST API**:

API specification in [openapi.yml](resources/openapi.yml) and code automatically generated with [oapi-codegen](https://github.com/deepmap/oapi-codegen).

> [!NOTE]
> Swagger UI is available at [/swagger/index.html](http://localhost:3001/swagger/index.html).

- **gRPC API**:

The [gRPC](https://grpc.io/) API is defined in the folder [proto/](proto).
The code is generated using [buf](https://buf.build/). in the folder [./internal/api/grpc](./internal/api/grpc).

## 🛠️ Getting Started

Run the project by:

```bash
go run ./cmd/go-web-layout/.
```

Or you can run it with docker using:

```bash
make dr
```
