package tools

import (
  "github.com/microcosm-cc/bluemonday"
  "strings"
)

func GetPolicy() *bluemonday.Policy {
  return bluemonday.StrictPolicy()
}

func PrepareCheckMap(values []string) map[string]bool {
  valuesMap := map[string]bool{}
  for _, value := range values {
    if value != "" {
      valuesMap[strings.ToLower(value)] = true
    }
  }
  return valuesMap
}
