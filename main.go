package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

const (
	LibraryFile     = ".library.yaml"
	IncomingFolder  = "Incoming"
	StorageFolder   = "Storage"
	SortedFolder    = "Sorted"
	ConflictFolder  = "Conflicts"
	DuplicateFolder = "Duplicates"
	TrashbinFolder  = "Trashbin"
)

// Flow:
// 	Incoming/*			-(process)->			Storage/camera/date
//						-(process)->			Duplicates/camera/date
//						-(process)->			Conflicts/*
// 	Storage/camera/date	-(manual recognize)->	Unsorted/*
// 	Unsorted/*			-(manual sort)->		Sorted/*

func main() {
	if err := mainFunc(); err != nil {
		fmt.Printf("Failed: %#v", err)
		os.Exit(1)
	}
}

var stdout *bufio.Writer

func mainFunc() error {
	stdout = bufio.NewWriter(os.Stdout)
	defer stdout.Flush()

	storage := pflag.StringP("storage", "s", ".", "Path to picture storage")
	pflag.Parse()
	storageFilename := path.Join(*storage, LibraryFile)

	logcfg := zap.NewDevelopmentConfig()
	logcfg.OutputPaths = []string{"log.log"}
	logger, err := logcfg.Build()
	if err != nil {
		return xerrors.Errorf("logger: %w", err)
	}
	defer logger.Sync()
	sl := logger.Sugar()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// break
	var wg sync.WaitGroup
	done := make(chan struct{})
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		wg.Add(1)
		defer wg.Done()
		select {
		case <-ctx.Done():
		case <-done:
		case <-signals:
			cancel()
		}
	}()
	time.Sleep(10 * time.Second)

	// load db
	fmt.Fprintf(stdout, "Loading library...\n")
	var db Database
	if err := db.Read(storageFilename); err != nil {
		return xerrors.Errorf("database: %w", err)
	}
	fmt.Fprintf(stdout, "Loaded %d entries\n", len(db.contents))
	stdout.Flush()

	defer func() {
		fmt.Fprintf(stdout, "Sync db\n")
		if err := db.Sync(storageFilename); err != nil {
			sl.Errorf("Error sync database: %#v", err)
		}
	}()

	// list files
	input, total, errs := Walk(ctx, *storage, IncomingFolder, map[string]struct{}{})
	for path, err := range errs {
		sl.Errorf("Error reading incoming %q: %#v", path, err)
	}
	sorted, known, errs := Walk(ctx, *storage, SortedFolder, db.sorted)
	for path, err := range errs {
		sl.Errorf("Error reading sorted %q: %#v", path, err)
	}

	// process sorted files
	kreg, kdup, kconf := 0, 0, 0
	var d Metric
	for hash, records := range sorted {
		for _, rec := range records {
			// select {
			// case <-ctx.Done():
			// 	return xerrors.Errorf("context: %w", ctx.Err())
			// default:
			// }
			n := time.Now()
			stdout.Flush()
			db.index[rec.TargetPath()] = struct{}{}
			// will probably be overwritten if hash conflicts or duplicates...
			newPath := path.Join(*storage, SortedFolder, rec.Path)
			if old, ok := db.contents[hash]; ok {
				if eq, err := CompareGroup(old.Paths[0], newPath); err != nil {
					sl.Errorf("Error processing sorted: %#v", err)
				} else if eq {
					kdup++
					old.Paths = append(old.Paths, newPath)
					sl.Errorf("Duplicate: %q, %q", old.Paths[0], newPath)
				} else {
					kconf++
					sl.Errorf("Hash conflict: %q, %q", old.Paths[0], newPath)
				}
				continue
			}
			kreg++
			d.Add(float64(time.Since(n).Milliseconds()))
			fmt.Fprintf(stdout, "Total:\t%10d | reg:\t%10d | dup\t%10d | con\t%10d | avg time %2.3f\r",
				known, kreg, kdup, kconf, d.Avg()/1000.0)
			sl.Infof("Register: %q", newPath)
			db.contents[hash] = &Record{
				Hash:      hash,
				Paths:     []string{newPath},
				Camera:    rec.Camera,
				Timestamp: rec.Timestamp,
				Sorted:    true,
			}
		}
	}
	fmt.Fprintf(stdout, "Total:\t%10d | reg:\t%10d | dup\t%10d | con\t%10d\n",
		known, kreg, kdup, kconf)
	stdout.Flush()

	//counters
	newfiles, removed, duplicates, conflict := 0, 0, 0, 0

	// process incoming files
	for hash, files := range input {
		select {
		case <-ctx.Done():
			return xerrors.Errorf("context: %w", ctx.Err())
		default:
		}
		fmt.Fprintf(stdout, "Total:\t%10d | new:\t%10d | rm\t%10d | dup\t%10d | con\t%10d\r",
			total, newfiles, removed, duplicates, conflict)
		stdout.Flush()
		var filenames []string
		for _, f := range files {
			filenames = append(filenames, path.Join(*storage, IncomingFolder, f.Path))
		}

		// check equality
		eq, err := CompareGroup(filenames...)
		if err != nil {
			sl.Errorf("Error comparing incoming: %#v", err)
			continue
		}
		if !eq {
			// they're different - hash conflict
			// move them all to Conflicts as is
			// dont add to database:
			// deal with them and try again
			for _, f := range files {
				conflict++
				if err := Move(sl,
					path.Join(*storage, IncomingFolder, f.Path),
					path.Join(*storage, ConflictFolder, f.Path)); err != nil {
					sl.Errorf("Error moving: %#v", err)
				}
			}
			continue
		}

		// all are equal, and we dont keep them here
		// remove all but first
		for _, f := range files[1:] {
			removed++
			// if err := Del(sl,
			// 	path.Join(*storage, IncomingFolder),
			// 	path.Join(*storage, IncomingFolder, f.Path)); err != nil {
			// 	sl.Errorf("Error removing: %#v", err)
			// }
			if err := Move(sl,
				path.Join(*storage, IncomingFolder, f.Path),
				path.Join(*storage, TrashbinFolder, f.Path)); err != nil {
				sl.Errorf("Error removing: %#v", err)
			}
		}
		file := files[0]

		// check strict existence
		if rec, ok := db.contents[hash]; ok {
			// found
			eq, err := CompareGroup(
				rec.Paths[0],
				path.Join(*storage, IncomingFolder, file.Path))
			if err != nil {
				sl.Errorf("Error processing incoming: %#v", err)
				continue
			}
			if eq {
				// proper duplicate
				removed++
				// if err := Del(sl,
				// 	path.Join(*storage, IncomingFolder),
				// 	path.Join(*storage, IncomingFolder, file.Path)); err != nil {
				// 	sl.Errorf("Error removing: %#v", err)
				// }
				if err := Move(sl,
					path.Join(*storage, IncomingFolder, file.Path),
					path.Join(*storage, TrashbinFolder, file.Path)); err != nil {
					sl.Errorf("Error removing: %#v", err)
				}
				continue
			}
			// hash conflict
			conflict++
			if err := Move(sl,
				path.Join(*storage, IncomingFolder, file.Path),
				path.Join(*storage, ConflictFolder, file.Path)); err != nil {
				sl.Errorf("Error moving: %#v", err)
			}
			continue
		}

		// check target directory
		if _, ok := db.index[file.TargetPath()]; ok {
			// directory seen - move to duplicates
			duplicates++
			if err := Move(sl,
				path.Join(*storage, IncomingFolder, file.Path),
				path.Join(*storage, DuplicateFolder, file.Path)); err != nil {
				sl.Errorf("Error moving: %#v", err)
			}
			continue
		}
		// new file - move to target
		dup := 0
		var name string
		for {
			name = path.Join(*storage, StorageFolder, file.TargetPath(), file.TargetFile(dup))
			if _, err := os.Stat(name); err != nil {
				if os.IsNotExist(err) {
					break
				}
				sl.Errorf("Error stat new file: %#v", err)
			}
			dup++
		}
		newfiles++
		Move(sl,
			path.Join(*storage, IncomingFolder, file.Path),
			name)
		file.Paths = []string{name}
		file.Path = ""
		db.contents[hash] = &file
		// do not update index here!
	}
	fmt.Fprintf(stdout, "Total:\t%10d | new:\t%10d | rm\t%10d | dup\t%10d | con\t%10d\n",
		total, newfiles, removed, duplicates, conflict)
	stdout.Flush()

	close(done)
	wg.Wait()
	return nil
}
