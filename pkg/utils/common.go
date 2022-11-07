package utils

import "strings"

func PrepareBoolMap(keys []string) map[string]bool {
	m := map[string]bool{}
	for _, key := range keys {
		if strings.TrimSpace(key) != "" {
			m[strings.ToLower(key)] = true
		}
	}
	return m
}
