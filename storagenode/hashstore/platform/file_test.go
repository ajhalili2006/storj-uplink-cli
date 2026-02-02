// Copyright (C) 2025 Storj Labs, Inc.
// See LICENSE for copying information.

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zeebo/assert"
)

func TestRemoveWithOtherProcessOpenHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testfile")

	// create the test file.
	fh, err := CreateFile(path)
	assert.NoError(t, err)
	assert.NoError(t, fh.Close())

	// construct a command to hold open the file and wait for signals.
	cmd := exec.Command("go", "run", "./testdata/fileop", "open-and-wait", path)
	stdout, err := cmd.StdoutPipe()
	assert.NoError(t, err)
	stdin, err := cmd.StdinPipe()
	assert.NoError(t, err)

	// start the process and set up cleanup if anything fails.
	assert.NoError(t, cmd.Start())
	defer func() { _ = cmd.Wait() }()
	defer func() { _ = stdin.Close() }()
	defer func() { _ = stdout.Close() }()

	// wait for the process to have the file open.
	_, err = stdout.Read(make([]byte, 1))
	assert.NoError(t, err)

	// remove the path.
	assert.NoError(t, os.Remove(path))

	// signal the other process to exit and wait for it.
	assert.NoError(t, stdin.Close())
	assert.NoError(t, cmd.Wait())

	// the file should be gone.
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestOtherProcessRemoveWithOpenHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testfile")

	remove := func() {
		cmd := exec.Command("go", "run", "./testdata/fileop", "remove", path)
		output, err := cmd.CombinedOutput()
		assert.Equal(t, string(output), "")
		assert.NoError(t, err)
	}

	t.Run("CreateHandle", func(t *testing.T) {
		fh, err := CreateFile(path)
		assert.NoError(t, err)
		defer func() { _ = fh.Close() }()

		remove()

		assert.NoError(t, fh.Close())
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("ReadHandle", func(t *testing.T) {
		fh, err := CreateFile(path)
		assert.NoError(t, err)
		assert.NoError(t, fh.Close())

		fh, err = OpenFileReadOnly(path)
		assert.NoError(t, err)
		defer func() { _ = fh.Close() }()

		remove()

		assert.NoError(t, fh.Close())
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("ReadWriteHandle", func(t *testing.T) {
		fh, err := CreateFile(path)
		assert.NoError(t, err)
		assert.NoError(t, fh.Close())

		fh, err = OpenFileReadWrite(path)
		assert.NoError(t, err)
		defer func() { _ = fh.Close() }()

		remove()

		assert.NoError(t, fh.Close())
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})
}
