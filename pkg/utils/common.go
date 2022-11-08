package utils

import "strings"

func PrepareBoolMap(keys []string, caseInsensitive bool) map[string]bool {
  m := map[string]bool{}
  for _, key := range keys {
    key = strings.TrimSpace(key)
    if key == "" {
      continue
    }
    if caseInsensitive {
      key = strings.ToLower(key)
    }
    m[key] = true
  }
  return m
}
