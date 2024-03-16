package main

import (
	"fmt"
	"os"
	"path"

	"go.uber.org/zap"
)

func Move(sl *zap.SugaredLogger, from, to string) error {
	dir, _ := path.Split(to)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("move %q: %w", dir, err)
	}
	sl.Debugf("move %q %q", from, to)
	return nil
}

func Del(sl *zap.SugaredLogger, root, f string) error {
	if err := os.Remove(f); err != nil {
		return fmt.Errorf("rm file %q: %w", f, err)
	}
	// todo: remove empty dirs
	sl.Debugf("delete %q", f)
	return nil
}
