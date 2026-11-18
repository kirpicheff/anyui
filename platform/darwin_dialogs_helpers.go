//go:build darwin

package platform

import "strings"

// extractDarwinFileExtensions извлекает расширения без точки и звёздочки из FileFilter.
func extractDarwinFileExtensions(filters []FileFilter) []string {
	seen := make(map[string]struct{})
	var res []string
	for _, f := range filters {
		for _, token := range strings.Split(f.Pattern, ";") {
			token = strings.TrimSpace(token)
			token = strings.TrimPrefix(token, "*.")
			token = strings.TrimPrefix(token, ".")
			if token == "" || token == "*" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			res = append(res, token)
		}
	}
	return res
}

