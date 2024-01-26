package main

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/xerrors"
)

const chunkSize = 1024 * 100

func DeepCompareFiles(file1, file2 string) (bool, error) {
	// check stats first
	info1, err := os.Stat(file1)
	if err != nil {
		return false, xerrors.Errorf("stat %s: %w", file1, err)
	}
	info2, err := os.Stat(file1)
	if err != nil {
		return false, xerrors.Errorf("stat %s: %w", file2, err)
	}
	if info1.Size() != info2.Size() {
		return false, nil
	}

	// check contents
	f1, err := os.Open(file1)
	if err != nil {
		return false, xerrors.Errorf("open %s: %w", file1, err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		return false, xerrors.Errorf("open %s: %w", file2, err)
	}
	defer f2.Close()

	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)
	for {
		n1, err1 := f1.Read(b1)
		if err1 != nil && err1 != io.EOF {
			return false, xerrors.Errorf("read %s: %w", file1, err1)
		}

		n2, err2 := f2.Read(b2)
		if err2 != nil && err2 != io.EOF {
			return false, xerrors.Errorf("read %s: %w", file2, err2)
		}
		if n1 != n2 {
			return false, nil
		}

		switch {
		case err1 == io.EOF && err2 == io.EOF:
			return true, nil
		case err1 == io.EOF || err2 == io.EOF:
			return false, nil
		}

		if !bytes.Equal(b1, b2) {
			return false, nil
		}
	}
}

func CompareGroup(files ...string) (bool, error) {
	if len(files) <= 1 {
		return true, nil
	}
	// check if they're equal
	for _, f := range files[1:] {
		if eq, err := DeepCompareFiles(files[0], f); err != nil {
			return false, xerrors.Errorf("compare %s, %s: %w", files[0], f, err)
		} else if !eq {
			return false, nil
		}
	}
	return true, nil
}
