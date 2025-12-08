package goweblayout

//go:generate go tool oapi-codegen -config ./resources/openapi-cfg.yaml ./resources/openapi.yml
//go:generate go tool gospecpaths --package rest --output ./internal/api/rest/paths.gen.go ./resources/openapi.yml
//go:generate sqlc generate -f ./sqlc.yml
