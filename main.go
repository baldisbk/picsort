package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/spf13/pflag"
)

func main() {
	// importMode := pflag.BoolP("import", "i", false, "Import existing files, leave them in place")
	// addMode := pflag.BoolP("add", "a", false, "Add files, move them to storage")
	// lib := pflag.StringP("lib", "l", "library.yaml", "Path to library file")
	// storage := pflag.StringP("storage", "s", "", "Path to picture storage")
	pflag.Parse()
	inputs := pflag.Args()

	files := []string{}
	for _, input := range inputs {
		filesys := os.DirFS(input)
		fs.WalkDir(filesys, ".", func(p string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				files = append(files, path.Join(input, p))
			}
			return nil
		})
	}
	fmt.Println(files)
}
