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

func AtLeastOneEqual(first, second []string) bool {
  if len(first) > len(second) {
    first, second = second, first
  }
  for idx, valueFirst := range first {
    valueSecond := second[idx]
    if valueFirst == valueSecond {
      return true
    }
  }
  return false
}
