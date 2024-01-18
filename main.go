package main

import (
  "flag"
  "log"
  "main/suggest"
  "main/suggest_merger"
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

func RunServingSuggestMerger(mergerConfigPath, port string) {
  if mergerConfigPath == "" {
    log.Fatalln("please specify the merger config data path via the --merger-config parameter")
    return
  }

  mergerConfig, _ := suggest_merger.ReadConfig(mergerConfigPath)
  if len(mergerConfig.SuggestShardsUrls) == 0 {
    log.Fatalln("urls not found in merger-config")
    return
  }

  mh, err := suggest_merger.NewHandler(mergerConfig)
  if err != nil {
    log.Fatalln(err)
    return
  }

  log.Println("merger ready to serve")

  http.Handle("/suggest", http.HandlerFunc(mh.HandleMergerSuggestRequest))
  http.Handle("/health", http.HandlerFunc(mh.HandleMergerHealthRequest))
  http.Handle("/", http.HandlerFunc(mh.HandleMergerHealthRequest))

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
  countWorkers := flag.Int("parallel", 5, "number of parallel workers")
  suffixSuggestFactor := flag.Float64("suffix-factor", 1e-5, "a weight multiplier for the suffix suggest")
  equalShapedNormalize := flag.Bool("equal-shaped-normalize", false, "additional normalization for cyrillic symbols")
  buildWithoutSuffixes := flag.Bool("build-without-suffixes", false, "build suggest without suffixes")
  countOutputFiles := flag.Int("count-output-files", 0, "build suggest to N result files")
  workAsMerger := flag.Bool("merger-on", false, "run suggest as merger")
  mergerConfigPath := flag.String("merger-config", "", "configuration for merger mode")

  port := flag.String("port", "8080", "daemon port")
  flag.Parse()

  if *suggestDataPath == "" && !*workAsMerger {
    log.Fatalln("please specify the suggest data path via the --suggest parameter")
  }
  if *inputFilePath != "" {
    if *countOutputFiles == 0 {
      suggest.DoBuildSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor, *buildWithoutSuffixes)
    } else {
      suggest_merger.DoBuildShardedSuggest(*inputFilePath, *suggestDataPath, *maxItemsPerPrefix, *suffixSuggestFactor, *buildWithoutSuffixes, *countOutputFiles, *countWorkers)
    }
    return
  }

  if *workAsMerger {
    RunServingSuggestMerger(*mergerConfigPath, *port)
  } else {
    RunServingSuggest(*suggestDataPath, *port, *equalShapedNormalize)
  }

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
