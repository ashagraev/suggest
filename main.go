package main

import (
  "flag"
  "github.com/microcosm-cc/bluemonday"
  "google.golang.org/protobuf/proto"
  "io/ioutil"
  "log"
  "net/http"
)

func getPolicy() *bluemonday.Policy {
  return bluemonday.StrictPolicy()
}

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
  log.Printf("marshalling suggest as proto")
  b, err := proto.Marshal(suggestData)
  if err != nil {
    log.Fatalln(err)
  }
  log.Printf("writing the resulting proto suggest data to %s", suggestDataPath)
  if err := ioutil.WriteFile(suggestDataPath, b, 0644); err != nil {
    log.Fatalln(err)
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

  suggestData, err := LoadSuggest(*suggestDataPath)
  if err != nil {
    log.Fatalln(err)
  }
  h := &Handler{
    Suggest:              suggestData,
    Policy:               getPolicy(),
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
