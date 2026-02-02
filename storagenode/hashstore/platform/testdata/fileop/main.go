// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"fmt"
	"os"

	"storj.io/storj/storagenode/hashstore/platform"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage: fileop <op> <path>")
	}
	op, path := os.Args[1], os.Args[2]

	switch op {
	case "remove":
		if err := os.Remove(path); err != nil {
			panic(err)
		}
		os.Exit(0)
	case "open-and-wait":
		openHandleAndWait(path)
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "fileop: unknown operation", op)
		os.Exit(1)
	}
}

func openHandleAndWait(path string) {
	fh, err := platform.OpenFileReadOnly(path)
	if err != nil {
		panic(err)
	}
	defer func() { _ = fh.Close() }()

	_, _ = os.Stdout.Write(make([]byte, 1)) // signal that the file is open
	_, _ = os.Stdin.Read(make([]byte, 1))   // wait for stdin to be closed
}
