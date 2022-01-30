package main

import (
  "github.com/microcosm-cc/bluemonday"
  "regexp"
  "strings"
)

var alphaRegExp = regexp.MustCompile(`[a-zA-Z0-9]+`)

func NormalizeString(s string, p *bluemonday.Policy) string {
  s = p.Sanitize(s)
  s = strings.ToLower(s)
  s = strings.Join(alphaRegExp.FindAllString(s, -1), " ")
  return s
}
