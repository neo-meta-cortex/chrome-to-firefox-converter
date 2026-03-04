package js

import (
	"fmt"
	"os"
	"strings"
)

// knownUnsupported lists Chrome APIs with no Firefox equivalent or limited support.
var knownUnsupported = []string{
	"chrome.enterprise",
	"chrome.certificateProvider",
	"chrome.documentScan",
	"chrome.fileBrowserHandler",
	"chrome.fileSystemProvider",
	"chrome.loginState",
	"chrome.platformKeys",
	"chrome.printingMetrics",
	"chrome.wallpaper",
}

// TransformResult holds the result of transforming a single JS file.
type TransformResult struct {
	FilePath    string
	Replacements int
	Warnings    []string
}

// TransformFile reads a JS file, replaces chrome.* to browser.*, and writes it to dstPath.
func TransformFile(srcPath, dstPath string) (TransformResult, error) {
	result := TransformResult{FilePath: srcPath}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return result, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)

	// Check for unsupported APIs before replacing
	for _, api := range knownUnsupported {
		if strings.Contains(content, api) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Unsupported API detected: %s (no Firefox equivalent)", api))
		}
	}

	// Count replacements before doing them
	result.Replacements = strings.Count(content, "chrome.")

	// Core replacement: chrome. to browser.
	// We use a careful replacement that avoids hitting things like
	// "chrome-extension://" URLs or CSS color names etc.
	content = replaceChromeCalls(content)

	if err := os.WriteFile(dstPath, []byte(content), 0644); err != nil {
		return result, fmt.Errorf("writing file: %w", err)
	}

	return result, nil
}

// replaceChromeCalls does a context-aware replacement of chrome.* API calls.
func replaceChromeCalls(content string) string {
	var sb strings.Builder
	i := 0

	for i < len(content) {
		// Look for "chrome."
		idx := strings.Index(content[i:], "chrome.")
		if idx == -1 {
			sb.WriteString(content[i:])
			break
		}

		// Write everything before the match
		sb.WriteString(content[i : i+idx])

		// Check what comes after "chrome." — if it's a letter (API call), replace it.
		// Skip things like "chrome-extension://" or "chromebook"
		after := content[i+idx+7:] // skip "chrome."
		if len(after) > 0 && isAPIChar(after[0]) {
			sb.WriteString("browser.")
		} else {
			sb.WriteString("chrome.")
		}

		i = i + idx + 7
	}

	return sb.String()
}

func isAPIChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}
