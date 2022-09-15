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

func Equal[T comparable](first, second []T) bool {
  if len(first) != len(second) {
    return false
  }
  for idx, valueFirst := range first {
    valueSecond := second[idx]
    if valueFirst != valueSecond {
      return false
    }
  }
  return true
}
