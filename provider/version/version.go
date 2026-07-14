package version

// Version is set at build time via -ldflags by GoReleaser.
// It is the empty string in local dev builds.
var Version string
