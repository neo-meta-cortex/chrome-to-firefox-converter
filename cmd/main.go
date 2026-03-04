package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neo-meta-cortex/chrome-to-firefox-converter/internal/crx"
	"github.com/neo-meta-cortex/chrome-to-firefox-converter/internal/packager"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		os.Exit(0)
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: please provide both input and output paths.")
		fmt.Fprintln(os.Stderr, "Run `ctfextension -h` for usage information.")
		os.Exit(1)
	}

	src := args[0]
	dst := args[1]

	createXPI := false
	for _, a := range args[2:] {
		if a == "--xpi" {
			createXPI = true
		}
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: source path does not exist: %s\n", src)
		os.Exit(1)
	}

	fmt.Printf("\nChrome to Firefox Extension Converter\n")

	// If the input is a .crx file, extract it to a temp directory first
	if crx.IsCRX(src) {
		fmt.Printf("   Input:  %s (CRX archive)\n", src)

		tmpDir, err := os.MkdirTemp("", "ctfextension-crx-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not create temp dir: %v\n", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("   Extracting CRX...\n")
		if err := crx.Extract(src, tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to extract .crx file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   Extracted OK\n")

		src = tmpDir
	} else {
		fmt.Printf("   Input:  %s\n", src)
	}

	fmt.Printf("   Output: %s\n", dst)
	if createXPI {
		fmt.Printf("   XPI:    yes\n")
	}
	fmt.Println()

	ch := make(chan packager.Progress, 200)

	go func() {
		err := packager.Convert(packager.Options{
			SrcDir:    src,
			DstDir:    dst,
			CreateXPI: createXPI,
			Progress:  ch,
		})
		if err != nil {
			ch <- packager.Progress{Error: err}
		}
		close(ch)
	}()

	warnings := 0
	for p := range ch {
		if p.Error != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n", p.Error)
			os.Exit(1)
		}
		if p.Warning != "" {
			fmt.Printf("WARNING: %s\n", p.Warning)
			warnings++
		}
		if p.Message != "" {
			fmt.Printf("   %s\n", p.Message)
		}
	}

	fmt.Println()
	if warnings > 0 {
		fmt.Printf("Done - %d warning(s). Check output before publishing.\n", warnings)
	} else {
		fmt.Println("Done!")
	}

	_ = filepath.Clean(dst)
}

func printHelp() {
	fmt.Print(`
Chrome to Firefox Extension Converter

USAGE
  ctfextension <input-path> <output-path> [flags]

ARGUMENTS
  input-path     Path to a Chrome extension, either:
                   - An unpacked extension folder
                   - A .crx file (downloaded from Chrome Web Store)
  output-path    Path where the converted extension will be written
                 (created automatically if it does not exist)

FLAGS
  --xpi          Package the output as a .xpi file ready to
                 install in Firefox
  -h, --help     Show this help message

MORE INFO
  https://github.com/neo-meta-cortex/chrome-to-firefox-converter

`)
}
