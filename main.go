package main

import (
  "flag"
  "github.com/microcosm-cc/bluemonday"
  "log"
  "net/http"
)

func main() {
  inputFilePath := flag.String("input", "", "input data file path")
  maxItemsPerPrefix := flag.Int("count", 10, "number of suggestions to return")
  suffixSuggestFactor := flag.Float64("suffix-factor", 1e-5, "a weight multiplier for the suffix suggest")
  equalShapedNormalize := flag.Bool("equal-shaped-normalize", false, "additional normalization for cyrillic symbols")
  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  policy := bluemonday.StrictPolicy()
  items, err := LoadItems(*inputFilePath, policy)
  if err != nil {
    log.Fatalln(err)
  }

  suggest := BuildSuggest(items, *maxItemsPerPrefix, float32(*suffixSuggestFactor))
  h := &Handler{
    Suggest:              suggest,
    Policy:               policy,
    EqualShapedNormalize: *equalShapedNormalize,
  }
  log.Println("ready to serve")

  http.Handle("/suggest", http.HandlerFunc(h.HandleSuggestRequest))
  http.Handle("/health", http.HandlerFunc(h.HandleHealthRequest))
  http.Handle("/", http.HandlerFunc(h.HandleHealthRequest))

  err = http.ListenAndServe(":"+*port, nil)
  if err != nil {
    log.Fatalf("fatal error in ListenAndServe: %v", err)
  }
}
