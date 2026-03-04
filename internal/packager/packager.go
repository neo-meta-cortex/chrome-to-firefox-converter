package packager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neo-meta-cortex/chrome-to-firefox-converter/internal/js"
	"github.com/neo-meta-cortex/chrome-to-firefox-converter/internal/manifest"
)

// Progress is sent on the channel during conversion to update the TUI.
type Progress struct {
	Message  string
	Warning  string
	Error    error
	Done     bool
	FilesTotal     int
	FilesProcessed int
}

// Options configures the conversion.
type Options struct {
	SrcDir  string
	DstDir  string
	CreateXPI bool
	Progress chan<- Progress
}

// Convert runs the full conversion pipeline.
func Convert(opts Options) error {
	ch := opts.Progress

	// Count total files first
	total := 0
	filepath.Walk(opts.SrcDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total++
		}
		return nil
	})

	send(ch, Progress{Message: fmt.Sprintf("Found %d files to process", total), FilesTotal: total})

	// Clean and create output dir
	if err := os.RemoveAll(opts.DstDir); err != nil {
		return fmt.Errorf("clearing output dir: %w", err)
	}
	if err := os.MkdirAll(opts.DstDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	processed := 0

	err := filepath.Walk(opts.SrcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path and destination
		rel, _ := filepath.Rel(opts.SrcDir, srcPath)
		dstPath := filepath.Join(opts.DstDir, rel)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		processed++
		send(ch, Progress{
			Message:        fmt.Sprintf("Processing: %s", rel),
			FilesTotal:     total,
			FilesProcessed: processed,
		})

		switch {
		case filepath.Base(srcPath) == "manifest.json":
			warnings, err := manifest.Transform(srcPath, dstPath)
			if err != nil {
				return fmt.Errorf("manifest transform: %w", err)
			}
			for _, w := range warnings {
				send(ch, Progress{Warning: w})
			}

		case isJSFile(srcPath):
			result, err := js.TransformFile(srcPath, dstPath)
			if err != nil {
				return fmt.Errorf("js transform %s: %w", rel, err)
			}
			if result.Replacements > 0 {
				send(ch, Progress{
					Message: fmt.Sprintf("  Replaced %d chrome.* call(s) in %s", result.Replacements, rel),
				})
			}
			for _, w := range result.Warnings {
				send(ch, Progress{Warning: fmt.Sprintf("[%s] %s", rel, w)})
			}

		default:
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying %s: %w", rel, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	send(ch, Progress{Message: "Conversion complete!"})

	// Optionally create .xpi
	if opts.CreateXPI {
		xpiPath := opts.DstDir + ".xpi"
		send(ch, Progress{Message: fmt.Sprintf("Creating %s...", filepath.Base(xpiPath))})
		if err := createXPI(opts.DstDir, xpiPath); err != nil {
			return fmt.Errorf("creating xpi: %w", err)
		}
		send(ch, Progress{Message: fmt.Sprintf("XPI created: %s", xpiPath)})
	}

	send(ch, Progress{Done: true, FilesTotal: total, FilesProcessed: processed})
	return nil
}

func isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".js" || ext == ".mjs" || ext == ".ts"
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func createXPI(srcDir, xpiPath string) error {
	f, err := os.Create(xpiPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)

		fw, err := w.Create(rel)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = fw.Write(data)
		return err
	})
}

func send(ch chan<- Progress, p Progress) {
	if ch != nil {
		ch <- p
	}
}
