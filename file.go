package main

import (
	"os"
	"path"

	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

func Move(sl *zap.SugaredLogger, from, to string) error {
	dir, _ := path.Split(to)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return xerrors.Errorf("mkdir %q: %w", dir, err)
	}
	if err := os.Rename(from, to); err != nil {
		return xerrors.Errorf("move %q: %w", dir, err)
	}
	sl.Infof("move %q %q", from, to)
	return nil
}

func Del(sl *zap.SugaredLogger, root, f string) error {
	if err := os.Remove(f); err != nil {
		return xerrors.Errorf("rm file %q: %w", f, err)
	}
	// todo: remove empty dirs
	sl.Infof("delete %q", f)
	return nil
}
