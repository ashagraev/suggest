package tools

import (
  "github.com/microcosm-cc/bluemonday"
  "regexp"
  "strings"
)

var alphaRegExp = regexp.MustCompile(`[a-zA-Z]+|[0-9]+`)

func NormalizeString(s string, p *bluemonday.Policy) string {
  s = p.Sanitize(s)
  s = strings.ToLower(s)
  s = strings.Join(alphaRegExp.FindAllString(s, -1), " ")
  return s
}

func AlphaNormalizeString(s string) string {
  s = strings.Join(alphaRegExp.FindAllString(s, -1), " ")
  return s
}

func ToEqualShapedLatin(s string) string {
  rules := map[string]string{
    "У": "Y",
    "К": "K",
    "Е": "E",
    "Н": "H",
    "Х": "X",
    "В": "B",
    "А": "A",
    "п": "n",
    "Р": "P",
    "О": "O",
    "С": "C",
    "М": "M",
    "Т": "T",
    "З": "3",
  }
  for k, v := range rules {
    s = strings.ReplaceAll(s, k, v)
    s = strings.ReplaceAll(s, strings.ToLower(k), strings.ToLower(v))
  }
  return s
}

func EqualShapedNormalizeString(s string, p *bluemonday.Policy) string {
  s = p.Sanitize(s)
  s = strings.ToLower(s)
  s = ToEqualShapedLatin(s)
  s = strings.Join(alphaRegExp.FindAllString(s, -1), " ")
  return s
}
