package main

import (
	"fmt"
	"os"
	"path"

	"golang.org/x/xerrors"
)

func Move(from, to string) error {
	dir, _ := path.Split(to)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return xerrors.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.Rename(from, to); err != nil {
		return xerrors.Errorf("move %s: %w", dir, err)
	}
	fmt.Fprintf(os.Stdout, "delete\n\t%s\n\t%s\n", from, to)
	return nil
}

func Del(root, f string) error {
	if err := os.Remove(f); err != nil {
		return xerrors.Errorf("rm file %s: %w", f, err)
	}
	fmt.Fprintf(os.Stdout, "delete\n\t%s\n", f)
	return nil
}
