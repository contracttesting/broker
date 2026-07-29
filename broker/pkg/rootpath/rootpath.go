package rootpath

import (
	"os"
	"path/filepath"
)

func Discover() string {
	dir, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	return StartFrom(dir)
}

func StartFrom(start string) string {
	dir := start

	for {
		if dir == "/" {
			panic("root path not found")
		}

		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		dir = filepath.Dir(dir)
	}
}
