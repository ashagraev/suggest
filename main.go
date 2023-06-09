package main

import (
  "flag"
  "log"
  "net/http"
  "os"
  "os/signal"
  "syscall"
)

func RunServingSuggest(suggestDataPath, port string, equalShapedNormalize bool) {
  suggestData, err := LoadSuggest(suggestDataPath)
  if err != nil {
    log.Fatalln(err)
  }
  h := &Handler{
    Suggest:              suggestData,
    Policy:               getPolicy(),
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
  disableNormalizedParts := flag.Bool("disable-normalized-parts", false, "build suggest without normalized parts")

  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  if *suggestDataPath == "" {
    log.Fatalln("please specify the suggest data path via the --suggest parameter")
  }
  if *inputFilePath != "" {
    DoBuildSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor)
    return
  }

  RunServingSuggest(*suggestDataPath, *port, *equalShapedNormalize)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
