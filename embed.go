package goweblayout

import "embed"

//go:embed resources/migrations/*
var MigrationsFolder embed.FS

//go:embed resources/openapi.yml
var OpenAPI []byte

//go:embed static/swagger-ui/*
var SwaggerUI embed.FS
