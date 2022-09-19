package main

import "strings"

func Contains(source []string, item string) bool {
  sourceMap := map[string]bool{}
  for _, value := range source {
    sourceMap[strings.ToLower(value)] = true
  }
  if _, ok := sourceMap[strings.ToLower(item)]; ok {
    return true
  }
  return false
}

func AtLeastOneEqual(first, second []string) bool {
  firstMap := map[string]bool{}
  for _, firstValue := range first {
    firstMap[strings.ToLower(firstValue)] = true
  }
  for _, secondValue := range second {
    if _, ok := firstMap[strings.ToLower(secondValue)]; ok {
      return true
    }
  }
  return false
}

func Equal(first, second []string) bool {
  if len(first) != len(second) {
    return false
  }
  for idx, firstValue := range first {
    secondValue := second[idx]
    if firstValue != secondValue {
      return false
    }
  }
  return true
}
