package crx

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// IsCRX returns true if the given path is a .crx file.
func IsCRX(path string) bool {
	return filepath.Ext(path) == ".crx"
}

// Extract unpacks a .crx file into destDir and returns the directory path.
// Supports both CRX2 and CRX3 formats.
func Extract(crxPath, destDir string) error {
	data, err := os.ReadFile(crxPath)
	if err != nil {
		return fmt.Errorf("reading crx file: %w", err)
	}

	// Validate magic bytes: "Cr24"
	if len(data) < 16 || string(data[0:4]) != "Cr24" {
		// Some .crx files are plain zips (unpacked extensions renamed)
		// Try treating it as a zip directly
		return extractZip(data, destDir)
	}

	version := binary.LittleEndian.Uint32(data[4:8])

	var zipData []byte
	switch version {
	case 2:
		zipData, err = extractCRX2(data)
	case 3:
		zipData, err = extractCRX3(data)
	default:
		return fmt.Errorf("unsupported CRX version: %d", version)
	}

	if err != nil {
		return err
	}

	return extractZip(zipData, destDir)
}

// extractCRX2 skips the CRX2 header and returns the zip payload.
//
// CRX2 format:
//   [0-3]   magic "Cr24"
//   [4-7]   version = 2
//   [8-11]  public key length
//   [12-15] signature length
//   [16+]   public key, signature, then zip data
func extractCRX2(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("crx2: file too short")
	}
	pubKeyLen := binary.LittleEndian.Uint32(data[8:12])
	sigLen := binary.LittleEndian.Uint32(data[12:16])
	headerLen := 16 + pubKeyLen + sigLen
	if int(headerLen) >= len(data) {
		return nil, fmt.Errorf("crx2: invalid header lengths")
	}
	return data[headerLen:], nil
}

// extractCRX3 skips the CRX3 header and returns the zip payload.
//
// CRX3 format:
//   [0-3]   magic "Cr24"
//   [4-7]   version = 3
//   [8-11]  header size (protobuf CrxFileHeader)
//   [12+]   protobuf header bytes, then zip data
func extractCRX3(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("crx3: file too short")
	}
	headerSize := binary.LittleEndian.Uint32(data[8:12])
	zipStart := 12 + headerSize
	if int(zipStart) >= len(data) {
		return nil, fmt.Errorf("crx3: invalid header size")
	}
	return data[zipStart:], nil
}

// extractZip unpacks zip bytes into destDir.
func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating dest dir: %w", err)
	}

	for _, f := range r.File {
		// Sanitize path to prevent zip-slip attacks
		target := filepath.Join(destDir, filepath.Clean("/"+f.Name))
		if !isWithin(destDir, target) {
			return fmt.Errorf("zip entry outside destination: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening zip entry %s: %w", f.Name, err)
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return fmt.Errorf("creating file %s: %w", target, err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("extracting %s: %w", f.Name, err)
		}
	}

	return nil
}

// isWithin checks that target is inside baseDir (zip-slip guard).
func isWithin(baseDir, target string) bool {
	base := filepath.Clean(baseDir) + string(os.PathSeparator)
	return len(target) >= len(base) && target[:len(base)] == base
}
