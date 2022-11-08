package suggest

import (
  "log"
  "main/pkg/utils"
  "os"
  "google.golang.org/protobuf/proto"
  stpb "main/proto"
  "fmt"
  "strings"
  "encoding/json"
)

func BuildSuggest(inputFilePath string, suggestDataPath string, maxItemsPerPrefix int, suffixFactor float64) {
  policy := utils.GetSanitizerPolicy()
  items, err := loadItems(inputFilePath, policy)
  if err != nil {
    log.Fatalln(err)
  }
  suggestData, err := buildSuggestData(items, maxItemsPerPrefix, float32(suffixFactor))
  if err != nil {
    log.Fatalln(err)
  }
  log.Printf("marshalling app as proto")
  b, err := proto.Marshal(suggestData)
  if err != nil {
    log.Fatalln(err)
  }
  log.Printf("writing the resulting proto app data to %s", suggestDataPath)
  if err := os.WriteFile(suggestDataPath, b, 0644); err != nil {
    log.Fatalln(err)
  }
}

func buildSuggestData(items []*Item, maxItemsPerPrefix int, postfixWeightFactor float32) (*stpb.SuggestData, error) {
  overheadItemsCount := maxItemsPerPrefix * 2
  builder := &TrieBuilder{}
  for idx, item := range items {
    itemClasses, err := extractItemClasses(item)
    if err != nil {
      return nil, fmt.Errorf("unable to extract item classes: %v", err)
    }
    builder.Add(0, item.NormalizedText, overheadItemsCount, &TrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
      Classes:      itemClasses,
    })
    parts := strings.Split(item.NormalizedText, " ")
    for i := 1; i < len(parts); i++ {
      builder.Add(0, strings.Join(parts[i:], " "), overheadItemsCount, &TrieItem{
        Weight:       item.Weight * postfixWeightFactor,
        OriginalItem: item,
        Classes:      itemClasses,
      })
    }
    if (idx+1)%100000 == 0 {
      log.Printf("added %d items of %d to app", idx+1, len(items))
    }
  }
  log.Printf("finalizing app")
  builder.Finalize(maxItemsPerPrefix)
  return transform(builder)
}

func extractItemClasses(item *Item) ([]string, error) {
  b, err := json.Marshal(item.Data)
  if err != nil {
    return nil, fmt.Errorf("cannot convert data to json: %v", err)
  }
  params := &TrieItemClasses{}
  if err := json.Unmarshal(b, params); err != nil {
    return nil, fmt.Errorf("cannot parse json data: %v", err)
  }
  classes := params.Classes
  deprecatedClass := params.Class

  if len(classes) == 0 && deprecatedClass != "" {
    return append(classes, deprecatedClass), nil
  }
  itemClassesMap := utils.PrepareBoolMap(classes, false)
  if _, ok := itemClassesMap[deprecatedClass]; !ok {
    classes = append(classes)
  }
  return classes, nil
}
