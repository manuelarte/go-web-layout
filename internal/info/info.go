package info

//nolint:gochecknoglobals // We want to be able to override these values with ldflags.
var (
	Branch    string    // Branch - The branch used to build.
	BuildTime string    // BuildTime - Timestamp that the build occurred
	BuildURL  string    // BuildURL - URL of the build
	CommitID  string    // CommitID - The SHA1 checksum of the commit
	Version   = "local" // Version - The version
)
