//go:build !cgo
// +build !cgo

package database

// This file ensures modernc.org/sqlite is used when CGO is disabled
import _ "modernc.org/sqlite"

