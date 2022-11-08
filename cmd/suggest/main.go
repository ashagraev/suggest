package main

import (
  "flag"
  "log"
  "main/internal/app/handler"
  "net/http"
  "main/internal/app/suggest"
  "main/pkg/utils"
)

func main() {
  inputFilePath := flag.String("input", "", "input data file path")
  suggestDataPath := flag.String("suggest", "", "suggest data file path")
  maxItemsPerPrefix := flag.Int("count", 10, "number of suggestions to return")
  suffixSuggestFactor := flag.Float64("suffix-factor", 1e-5, "a weight multiplier for the suffix app")
  equalShapedNormalize := flag.Bool("equal-shaped-normalize", false, "additional normalization for cyrillic symbols")
  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  if *suggestDataPath == "" {
    log.Fatalln("please specify the app data path via the --app parameter")
  }
  if *inputFilePath != "" {
    suggest.BuildSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor)
    return
  }

  suggestData, err := suggest.LoadSuggest(*suggestDataPath)
  if err != nil {
    log.Fatalln(err)
  }
  h := &handler.Handler{
    Suggest:              suggestData,
    Policy:               utils.GetSanitizerPolicy(),
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
