package main

import (
  "flag"
  "google.golang.org/protobuf/proto"
  "log"
  "net/http"
  "os"
  "os/signal"
  "syscall"
)

func doBuildSuggest(inputFilePath string, suggestDataPath string, maxItemsPerPrefix int, suffixFactor float64) {
  policy := getPolicy()
  items, err := LoadItems(inputFilePath, policy)
  if err != nil {
    log.Fatalln(err)
  }
  suggestData, err := BuildSuggest(items, maxItemsPerPrefix, float32(suffixFactor))
  if err != nil {
    log.Fatalln(err)
  }
  SetVersion(suggestData)
  log.Printf("marshalling suggest as proto")
  b, err := proto.Marshal(suggestData)
  if err != nil {
    log.Fatalln(err)
  }
  log.Printf("writing the resulting proto suggest data to %s", suggestDataPath)
  if err := os.WriteFile(suggestDataPath, b, 0644); err != nil {
    log.Fatalln(err)
  }
}

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
  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  if *suggestDataPath == "" {
    log.Fatalln("please specify the suggest data path via the --suggest parameter")
  }
  if *inputFilePath != "" {
    doBuildSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor)
    return
  }

  RunServingSuggest(*suggestDataPath, *port, *equalShapedNormalize)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
