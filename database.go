package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type Record struct {
	Hash string `yaml:"-"`

	Path      string    `yaml:"path,omitempty"`
	Paths     []string  `yaml:"paths"`
	Camera    string    `yaml:"camera"`
	Timestamp time.Time `yaml:"timestamp"`
	Sorted    bool      `yaml:"sorted"`
}

func (r *Record) TargetPath() string {
	return fmt.Sprintf("%s/%04d-%02d-%02d",
		r.Camera,
		r.Timestamp.Year(),
		r.Timestamp.Month(),
		r.Timestamp.Day())
}

func (r *Record) TargetFile(dup int) string {
	ext := path.Ext(r.Path)
	if dup == 0 {
		return fmt.Sprintf("%02d-%02d-%02d%s",
			r.Timestamp.Hour(),
			r.Timestamp.Minute(),
			r.Timestamp.Second(),
			ext)
	} else {
		return fmt.Sprintf("%02d-%02d-%02d-%d%s",
			r.Timestamp.Hour(),
			r.Timestamp.Minute(),
			r.Timestamp.Second(),
			dup, ext)
	}
}

type Database struct {
	contents map[string]*Record
	index    map[string]struct{}
	sorted   map[string]struct{}
}

func (db *Database) Read(filename string) error {
	db.contents = map[string]*Record{}
	db.index = map[string]struct{}{}
	db.sorted = map[string]struct{}{}

	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return xerrors.Errorf("open: %w", err)
	}
	contents, err := io.ReadAll(f)
	if err != nil {
		return xerrors.Errorf("read: %w", err)
	}
	if err := yaml.Unmarshal(contents, &db.contents); err != nil {
		return xerrors.Errorf("unmarshal: %w", err)
	}
	for _, f := range db.contents {
		db.index[f.TargetPath()] = struct{}{}
		paths := []string{}
		if f.Path != "" {
			paths = append(paths, f.Path)
		}
		for _, p := range f.Paths {
			if p != "" {
				paths = append(paths, p)
			}
		}
		f.Paths = paths

		f.Path = ""
		if f.Sorted {
			for _, p := range f.Paths {
				db.sorted[p] = struct{}{}
			}
		}
	}
	return nil
}

func (db *Database) Sync(filename string) error {
	contents, err := yaml.Marshal(db.contents)
	if err != nil {
		return xerrors.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(filename, contents, fs.FileMode(0666)); err != nil {
		return xerrors.Errorf("write: %w", err)
	}
	return nil
}
