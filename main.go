package main

import (
  "flag"
  "log"
  "main/suggest"
  "main/tools"
  "net/http"
  "os"
  "os/signal"
  "syscall"
)

func RunServingSuggest(suggestDataPath, port string, equalShapedNormalize bool) {
  suggestData, err := suggest.LoadSuggest(suggestDataPath)
  if err != nil {
    log.Fatalln(err)
  }
  h := &suggest.Handler{
    Suggest:              suggestData,
    Policy:               tools.GetPolicy(),
    EqualShapedNormalize: equalShapedNormalize,
  }

  log.Println("ready to serve")

  http.Handle("/suggest", http.HandlerFunc(h.HandleSuggestRequest))
  http.Handle("/health", http.HandlerFunc(h.HandleHealthRequest))
  http.Handle("/", http.HandlerFunc(h.HandleHealthRequest))

  go ContinuouslyServe(port)
}

func ContinuouslyServe(port string) {
  if err := http.ListenAndServe(":"+port, nil); err != nil {
    log.Fatalf("fatal error in ListenAndServe: %v", err)
  }
}

func main() {
  inputFilePath := flag.String("input", "", "input data file path")
  suggestDataPath := flag.String("suggest", "", "suggest data file path")
  maxItemsPerPrefix := flag.Int("count", 10, "number of suggestions to return")
  suffixSuggestFactor := flag.Float64("suffix-factor", 1e-5, "a weight multiplier for the suffix suggest")
  equalShapedNormalize := flag.Bool("equal-shaped-normalize", false, "additional normalization for cyrillic symbols")
  buildWithoutSuffixes := flag.Bool("build-without-suffixes", false, "build suggest without suffixes")

  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  if *suggestDataPath == "" {
    log.Fatalln("please specify the suggest data path via the --suggest parameter")
  }
  if *inputFilePath != "" {
    suggest.DoBuildSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor, *buildWithoutSuffixes)
    return
  }

  RunServingSuggest(*suggestDataPath, *port, *equalShapedNormalize)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
