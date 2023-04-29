package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	homedir, err := os.UserHomeDir()
	check(err)

	lilyfonts := filepath.Join(homedir, "/Programs/lilypond/share/lilypond/2.24.0/fonts")

	copyFiles("./fonts/*/otf/*.otf", filepath.Join(lilyfonts, "otf"))
	copyFiles("./fonts/*/svg/*.svg", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/svg/*.svg", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/svg/*.woff", filepath.Join(lilyfonts, "svg"))
	copyFiles("./fonts/*/woff/*.woff", filepath.Join(lilyfonts, "svg"))
}

func copyFiles(glob, targetDir string) {
	files, err := filepath.Glob("./fonts/*/otf/*.otf")
	check(err)

	for _, file := range files {
		// TODO: make cross-platform
		_, err := exec.Command("cp", file, filepath.Join(targetDir, filepath.Base(file))).CombinedOutput()
		check(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
