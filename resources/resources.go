// Package resources contains all the resources for the application that can be embedded.
package resources

import "embed"

//go:embed migrations/*
var MigrationsFolder embed.FS

//go:embed openapi.yml
var OpenAPI []byte
