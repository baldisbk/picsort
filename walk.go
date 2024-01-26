package main

import (
	"context"
	"io/fs"
	"os"
	"path"

	"golang.org/x/xerrors"
)

func Walk(ctx context.Context, root, dir string) (res map[string][]Record, errs map[string]error) {
	res = map[string][]Record{}
	errs = map[string]error{}
	prefix := path.Join(root, dir)
	fs.WalkDir(os.DirFS(prefix), ".", func(f string, d fs.DirEntry, errInp error) error {
		select {
		case <-ctx.Done():
			errs[f] = xerrors.Errorf("context: %w", ctx.Err())
			return nil
		default:
		}
		if errInp != nil {
			errs[f] = xerrors.Errorf("input: %w", errInp)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rec := Record{Path: f}
		filename := path.Join(prefix, f)
		cam, ts, err := ReadExif(filename)
		if err != nil {
			errs[f] = xerrors.Errorf("exif: %w", err)
			return nil
		}
		rec.Camera = cam
		rec.Timestamp = ts
		h, err := ReadHash(filename)
		if err != nil {
			errs[f] = xerrors.Errorf("hash: %w", err)
			return nil
		}
		rec.Hash = h

		res[h] = append(res[h], rec)

		return nil
	})
	return
}
