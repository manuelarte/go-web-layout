# Go Web Layout

[![Go](https://github.com/manuelarte/go-template/actions/workflows/go.yml/badge.svg)](https://github.com/manuelarte/go-template/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/manuelarte/go-template)](https://goreportcard.com/report/github.com/manuelarte/go-template)
![version](https://img.shields.io/github/v/release/manuelarte/go-template)

Example layout for a Go web application.

## 🚀 Features

### Database Migration with golang-migrate

This project is using [golang-migrate](https://github.com/golang-migrate/migrate) to manage database migrations.
Inside the folder [./resources/migrations](./resources/migrations) you can find the migrations files.
They are embedded in the binary using [go-embed](https://pkg.go.dev/embed).

### REST API Generation using oapi-codegen

This project is using [oapi-codegen](https://github.com/deepmap/oapi-codegen) to generate the REST API.
The API is defined in the file [openapi.yml](openapi.yml), and the generated code is in the folder [./internal/api/rest](./internal/api/rest).

### GRPC Api Generation using protoc-gen-go and buf

This project is using [buf](https://buf.build/) to generate the GRPC API.
The GRPC API is defined in the folder [proto/](proto/), and the generated code is in the folder [./internal/api/grpc](./internal/api/grpc).
