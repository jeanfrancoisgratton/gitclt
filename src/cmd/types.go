// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/types.go
// Original timestamp : 2026.10.18 00:48:22

package cmd

// buildVersion and buildDate are injected at link time via
// `-ldflags "-X gitclt/cmd.buildVersion=... -X gitclt/cmd.buildDate=..."`
// by every build path (see version-bump.sh and each __*/{Makefile,APKBUILD,
// PKGBUILD,spec}); a plain `go build` leaves them at these fallbacks.
var buildVersion = "dev"
var buildDate = "unknown"
