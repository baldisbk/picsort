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
		return xerrors.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.Rename(from, to); err != nil {
		return xerrors.Errorf("move %s: %w", dir, err)
	}
	sl.Infof("move %s %s", from, to)
	return nil
}

func Del(sl *zap.SugaredLogger, root, f string) error {
	if err := os.Remove(f); err != nil {
		return xerrors.Errorf("rm file %s: %w", f, err)
	}
	// todo: remove empty dirs
	sl.Infof("delete %s", f)
	return nil
}
