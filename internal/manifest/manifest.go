package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// chromeOnlyPermissions are permissions that exist in Chrome but not Firefox.
// They will be stripped from the manifest and a warning issued for each.
var chromeOnlyPermissions = map[string]bool{
	"sidePanel":      true,
	"debugger":       true,
	"system.display": true,
	"offscreen":      true,
	"tabCapture":     true,
	"pageCapture":    true,
	"enterprise":     true,
	"signin":         true,
	"audio":          true,
	"transientBackground": true,
}

// chromeOnlyFields are top-level manifest keys that Firefox does not recognise.
// They will be removed from the manifest and a warning issued for each.
var chromeOnlyFields = []string{
	"key",
	"update_url",
	"externally_connectable",
	"platforms",
	"oauth2",
	"file_browser_handlers",
	"input_components",
	"automation",
	"sandbox",
}

// Transform reads a Chrome manifest.json, converts it for Firefox, and writes it to dstPath.
func Transform(srcPath, dstPath string) ([]string, error) {
	var warnings []string

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	// Detect manifest version
	mv := 2
	if v, ok := m["manifest_version"].(float64); ok {
		mv = int(v)
	}

	// Add browser_specific_settings (gecko ID)
	name := "converted-extension"
	if n, ok := m["name"].(string); ok {
		name = strings.ToLower(strings.ReplaceAll(n, " ", "-"))
	}
	m["browser_specific_settings"] = map[string]any{
		"gecko": map[string]any{
			"id":                 fmt.Sprintf("%s@converted.firefox", name),
			"strict_min_version": "109.0",
		},
	}

	// MV3 to MV2 conversions
	if mv == 3 {
		m["manifest_version"] = 2
		warnings = append(warnings, "Manifest V3 detected - converting to V2 for Firefox compatibility")

		// action to browser_action
		if action, ok := m["action"]; ok {
			m["browser_action"] = action
			delete(m, "action")
			warnings = append(warnings, "Renamed 'action' to 'browser_action'")
		}

		// service_worker to background scripts
		if bg, ok := m["background"].(map[string]any); ok {
			if sw, ok := bg["service_worker"].(string); ok {
				m["background"] = map[string]any{
					"scripts": []string{sw},
				}
				warnings = append(warnings, fmt.Sprintf("Converted service_worker '%s' to background script", sw))
			}
		}

		// host_permissions merged into permissions
		if hp, ok := m["host_permissions"].([]any); ok {
			existing := []any{}
			if p, ok := m["permissions"].([]any); ok {
				existing = p
			}
			m["permissions"] = append(existing, hp...)
			delete(m, "host_permissions")
			warnings = append(warnings, "Merged 'host_permissions' into 'permissions'")
		}

		// web_accessible_resources: MV3 uses array of objects, MV2 uses array of strings
		if war, ok := m["web_accessible_resources"].([]any); ok {
			var resources []string
			for _, item := range war {
				if obj, ok := item.(map[string]any); ok {
					if res, ok := obj["resources"].([]any); ok {
						for _, r := range res {
							if s, ok := r.(string); ok {
								resources = append(resources, s)
							}
						}
					}
				}
			}
			if len(resources) > 0 {
				m["web_accessible_resources"] = resources
				warnings = append(warnings, "Converted 'web_accessible_resources' from MV3 to MV2 format")
			}
		}

		// content_security_policy: MV3 uses object, MV2 uses string
		if csp, ok := m["content_security_policy"].(map[string]any); ok {
			if ep, ok := csp["extension_pages"].(string); ok {
				m["content_security_policy"] = ep
				warnings = append(warnings, "Converted 'content_security_policy' from MV3 object to MV2 string")
			}
		}
	}

	// Strip Chrome-only permissions and warn for each one removed
	if perms, ok := m["permissions"].([]any); ok {
		var kept []any
		for _, p := range perms {
			if s, ok := p.(string); ok {
				if chromeOnlyPermissions[s] {
					warnings = append(warnings, fmt.Sprintf("Removed unsupported permission '%s' (Chrome-only, no Firefox equivalent)", s))
					continue
				}
				// Also warn about permissions with limited support but keep them
				if s == "declarativeNetRequest" {
					warnings = append(warnings, "Permission 'declarativeNetRequest' has limited Firefox support - test carefully")
				}
			}
			kept = append(kept, p)
		}
		m["permissions"] = kept
	}

	// Remove Chrome-only top-level manifest fields and warn for each one
	for _, field := range chromeOnlyFields {
		if _, exists := m[field]; exists {
			delete(m, field)
			warnings = append(warnings, fmt.Sprintf("Removed Chrome-only manifest field '%s' (not supported in Firefox)", field))
		}
	}

	// Write output
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serializing manifest: %w", err)
	}

	if err := os.WriteFile(dstPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	return warnings, nil
}
