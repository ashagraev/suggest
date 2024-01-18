package suggest_merger

import (
  "bufio"
  "fmt"
  "github.com/microcosm-cc/bluemonday"
  "google.golang.org/protobuf/proto"
  "io"
  "io/ioutil"
  "log"
  "main/suggest"
  "main/tools"
  "os"
  "sort"
  "strings"
  "time"
)

type characterStat struct {
  Count      int
  StartIndex int
  EndIndex   int
}

func DoBuildShardedSuggest(
  inputFilePath string,
  suggestDataPath string,
  maxItemsPerPrefix int,
  suffixFactor float64,
  buildWithoutSuffixes bool,
  countOutputFiles int,
  countWorkers int,
) {
  if !isFileSorted(inputFilePath) {
    log.Fatalf("file is not sorted, use linux command 'sort', example: sort --parallel 4 -o suggest.data suggest.data")
  }

  charactersStat, err := getCharacterStatByPrefixes(inputFilePath)
  if err != nil {
    log.Fatalln(err)
  }

  parts, err := getDistributionByParts(charactersStat, countOutputFiles)
  if err != nil {
    log.Fatalln(err)
  }

  policy := tools.GetPolicy()
  suggestVersion := uint64(time.Now().Unix())

  // start workers
  jobs := make(chan makeSuggestInput, countOutputFiles)
  errs := make(chan error, countOutputFiles)
  for i := 0; i < countWorkers; i++ {
    go makeSuggest(i+1,
      charactersStat,
      policy,
      suggestVersion,
      suggestDataPath,
      suffixFactor,
      maxItemsPerPrefix,
      buildWithoutSuffixes,
      inputFilePath,
      jobs, errs)
  }

  for shardNumber, characters := range parts {
    jobs <- makeSuggestInput{
      characters:  characters,
      shardNumber: shardNumber,
    }
  }
  close(jobs)

  for i := 0; i < len(parts); i++ {
    err := <-errs
    if err != nil {
      log.Fatalln(err)
    }
  }

  return
}

type makeSuggestInput struct {
  characters  []string
  shardNumber int
}

func makeSuggest(
  workerNumber int,
  charactersStat map[string]*characterStat,
  policy *bluemonday.Policy,
  suggestVersion uint64,
  suggestDataPath string,
  suffixFactor float64,
  maxItemsPerPrefix int,
  buildWithoutSuffixes bool,
  inputFilePath string,
  jobs <-chan makeSuggestInput,
  errs chan<- error,
) {
  for input := range jobs {
    characters := input.characters
    shardNumber := input.shardNumber

    var items []*suggest.Item

    for _, c := range characters {
      itemsPart, err := loadItemsByPart(workerNumber, inputFilePath, charactersStat[c].StartIndex, charactersStat[c].EndIndex, policy)
      if err != nil {
        errs <- err
      }
      items = append(items, itemsPart...)
    }

    log.Printf("worker #%d starts building suggest", workerNumber)
    suggestData, err := suggest.BuildSuggestData(items, maxItemsPerPrefix, float32(suffixFactor), buildWithoutSuffixes)
    if err != nil {
      errs <- err
    }
    suggest.SetVersion(suggestData, suggestVersion)

    log.Printf("worker #%d serializes suggest as proto", workerNumber)
    b, err := proto.Marshal(suggestData)
    if err != nil {
      errs <- err
    }

    suggestDataPathPart := strings.ReplaceAll(suggestDataPath, ".", fmt.Sprintf("_%d.", shardNumber))

    log.Printf("worker #%d writes the resulting proto suggest data to %s with prefixes %v, items count %d, version %d", workerNumber, suggestDataPathPart, characters, len(items), suggestData.Version)
    if err := ioutil.WriteFile(suggestDataPathPart, b, 0644); err != nil {
      errs <- err
    }

    errs <- nil
  }
}

func isFileSorted(inputFilePath string) bool {
  file, err := os.Open(inputFilePath)
  if err != nil {
    return false
  }
  defer file.Close()

  setOfFirstCharacters := make([]string, 0)

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    line := strings.TrimSpace(scanner.Text())

    if len(line) == 0 {
      continue
    }

    setOfFirstCharacters = append(setOfFirstCharacters, strings.ToLower(line[0:1]))
  }

  return sort.SliceIsSorted(setOfFirstCharacters, func(i, j int) bool {
    return setOfFirstCharacters[i] < setOfFirstCharacters[j]
  })
}

func getCharacterStatByPrefixes(inputFilePath string) (map[string]*characterStat, error) {
  symbolsMapCounter := map[string]*characterStat{}

  file, err := os.Open(inputFilePath)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  scanner := bufio.NewScanner(file)
  lineNumber := 0
  currentCharacterForStat := ""
  currentStartPos, currentEndPos, currentCounter := 0, 0, 0

  for scanner.Scan() {
    line := strings.TrimSpace(scanner.Text())

    if len(line) == 0 {
      continue
    }

    firstCharacter := strings.ToLower(line[0:1])

    if firstCharacter != currentCharacterForStat {
      if currentCharacterForStat != "" {
        currentEndPos += currentCounter - 1
        symbolsMapCounter[currentCharacterForStat] = &characterStat{
          Count:      currentCounter,
          StartIndex: currentStartPos,
          EndIndex:   currentEndPos,
        }
        currentStartPos = currentEndPos + 1
        currentEndPos = currentStartPos
        currentCounter = 0
      }
      currentCharacterForStat = firstCharacter
    }

    currentEndPos += len(line)
    currentCounter += 1

    lineNumber++
    if lineNumber%100000 == 0 {
      log.Printf("read %d lines", lineNumber)
    }
  }

  // processing for last characters
  if currentCharacterForStat != "" {
    currentEndPos += currentCounter - 1
    symbolsMapCounter[currentCharacterForStat] = &characterStat{
      Count:      currentCounter,
      StartIndex: currentStartPos,
      EndIndex:   currentEndPos,
    }
  }

  return symbolsMapCounter, nil
}

func getIndexOfMin(items []float64) int {
  min := items[0]
  minIdx := 0
  for i, item := range items {
    if item < min {
      min = item
      minIdx = i
    }
  }
  return minIdx
}

func getDistributionByParts(charactersStat map[string]*characterStat, countParts int) (map[int][]string, error) {
  characters := make([]string, 0, len(charactersStat))
  sumWeights := 0

  for c, value := range charactersStat {
    characters = append(characters, c)
    sumWeights += value.Count
  }
  sort.SliceStable(characters, func(i, j int) bool {
    return charactersStat[characters[i]].Count > charactersStat[characters[j]].Count
  })

  // the first estimate of the maximum part volume is the total volume divided to all parts
  maxSize := float64(sumWeights / countParts)

  // prepare array containing the current weight of the parts
  weightsParts := make([]float64, countParts)
  parts := map[int][]string{}
  restWeightsSum := sumWeights

  for _, c := range characters {
    weight := charactersStat[c].Count

    // put next value in part with lowest weight sum
    lowestPartIndex := getIndexOfMin(weightsParts)

    // calculate new weight of this part
    newWeightSum := weightsParts[lowestPartIndex] + float64(weight)
    foundPart := false
    for !foundPart {
      if newWeightSum <= maxSize {
        parts[lowestPartIndex] = append(parts[lowestPartIndex], c)
        weightsParts[lowestPartIndex] = newWeightSum
        restWeightsSum -= weight
        foundPart = true
      } else {
        // if not, increase the maxSize by the sum of the rest of the parts per part
        if restWeightsSum/countParts <= 1 {
          maxSize += float64(restWeightsSum)
        } else {
          maxSize += float64(restWeightsSum / countParts)
        }
      }
    }
  }
  return parts, nil
}

func loadItemsByPart(
  workerNumber int,
  inputFilePath string,
  startIndex int,
  endIndex int,
  policy *bluemonday.Policy,
) ([]*suggest.Item, error) {
  file, err := os.Open(inputFilePath)
  if err != nil {
    return nil, err
  }
  var items []*suggest.Item

  if _, err := file.Seek(int64(startIndex), io.SeekStart); err != nil {
    log.Fatal(err)
  }
  scanner := bufio.NewScanner(file)

  currentLen := startIndex
  lineNumber := 0
  for scanner.Scan() {
    line := strings.TrimSpace(scanner.Text())
    if len(line) == 0 {
      continue
    }
    item, err := suggest.NewItem(line, policy)
    if err != nil {
      return nil, fmt.Errorf("error processing line #%d: %v", lineNumber, err)
    }
    items = append(items, item)
    lineNumber++
    if lineNumber%100000 == 0 {
      log.Printf("worker #%d reads %d lines", workerNumber, lineNumber)
    }

    currentLen += len(line) + 1
    if currentLen >= endIndex {
      break
    }

  }
  return items, nil
}
