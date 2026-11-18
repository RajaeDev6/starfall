// Command assets draws every picture in assets/images and synthesizes every
// sound in assets/sounds. Run it from the repository root:
//
//	go run ./tools/assets
//
// The game only ever loads the files this writes, so to change a sprite or a
// sound, change it here and run this again.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	images := filepath.Join(root, "assets", "images")
	sounds := filepath.Join(root, "assets", "sounds")
	must(os.MkdirAll(images, 0o755))
	must(os.MkdirAll(sounds, 0o755))
	buildArt(images)
	buildSounds(sounds)
	fmt.Println("wrote", images, "and", sounds)
}
