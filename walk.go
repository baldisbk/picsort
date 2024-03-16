package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"
)

func Walk(ctx context.Context, root, dir string, visited map[string]struct{}) (res map[string][]Record, num int, errs map[string]error) {
	res = map[string][]Record{}
	errs = map[string]error{}
	prefix := path.Join(root, dir)
	var files []string
	fmt.Fprintf(stdout, "Listing %q...\n", dir)
	stdout.Flush()
	fs.WalkDir(os.DirFS(prefix), ".", func(f string, d fs.DirEntry, errInp error) error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if errInp != nil {
			errs[f] = fmt.Errorf("input: %w", errInp)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := visited[path.Join(prefix, f)]; ok {
			return nil
		}
		files = append(files, f)
		return nil
	})
	var d Metric
	fmt.Fprintf(stdout, "Found %d entries in %q, scanning...\n", len(files), dir)
	stdout.Flush()
	for _, f := range files {
		select {
		case <-ctx.Done():
			fmt.Println() // end progress
			return
		default:
		}
		n := time.Now()
		rec := Record{Path: f}
		filename := path.Join(prefix, f)
		cam, ts, err := ReadExif(filename)
		switch {
		case errors.Is(err, NotPictureErr):
			// empty hash - drop it
			// break
		case err != nil:
			errs[f] = fmt.Errorf("exif: %w", err)
			continue
		default:
			rec.Camera = cam
			rec.Timestamp = ts
			h, err := ReadHash(filename)
			if err != nil {
				errs[f] = fmt.Errorf("hash: %w", err)
				continue
			}
			rec.Hash = h
		}

		res[rec.Hash] = append(res[rec.Hash], rec)
		num++
		d.Add(float64(time.Since(n).Milliseconds()))
		fmt.Fprintf(stdout, "Scanned:\t%10d of %10d (avg time %2.3f)\r", num, len(files), d.Avg()/1000.0)
		stdout.Flush()
	}
	fmt.Println() // end progress
	return
}
