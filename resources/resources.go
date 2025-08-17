package resources

import "embed"

//go:embed migrations/*
var MigrationsFolder embed.FS
