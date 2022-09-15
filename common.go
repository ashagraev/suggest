package main

import "strings"

func PrepareCheckMap(values []string) map[string]bool {
  valuesMap := map[string]bool{}
  for _, value := range values {
    if value != "" {
      valuesMap[strings.ToLower(value)] = true
    }
  }
  return valuesMap
}
