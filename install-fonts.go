package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	homedir, err := os.UserHomeDir()
	check(err)

	var lilyfonts string
	switch runtime.GOOS {
	case "darwin":
		lilyfonts = filepath.Join(homedir, "Programs", "lilypond", "share", "lilypond", "2.24.0", "fonts")
	case "windows":
		lilyfonts = filepath.Join(homedir, ".local", "share", "fonts")
	default:
		panic("unhandled os " + runtime.GOOS)
	}

	copyFiles("./fonts/*/otf/*.otf", filepath.Join(lilyfonts, "otf"))
	copyFiles("./fonts/*/svg/*.svg", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/svg/*.svg", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/svg/*.woff", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/woff/*.woff", filepath.Join(lilyfonts, "svg"))
}

func copyFiles(glob, targetDir string) {
	files, err := filepath.Glob(glob)
	check(err)

	_ = os.MkdirAll(targetDir, 0755)

	for _, file := range files {
		data, err := os.ReadFile(file)
		check(err)

		err = os.WriteFile(filepath.Join(targetDir, filepath.Base(file)), data, 0644)
		check(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
