package main

import (
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/types/known/structpb"
  "io/ioutil"
  "log"
  stpb "main/proto/suggest/suggest_trie"
  "sort"
  "strings"
  "encoding/json"
  "fmt"
)

type SuggestionTextBlock struct {
  Text      string `json:"text"`
  Highlight bool   `json:"hl"`
}

type SuggestAnswerItem struct {
  Weight     float32                `json:"weight"`
  Data       map[string]interface{} `json:"data"`
  TextBlocks []*SuggestionTextBlock `json:"text"`
}

type SuggestTrieItemClasses struct {
  Classes []string `json:"classes"`
  Class   string   `json:"class"` // deprecated
}

type PaginatedSuggestResponse struct {
  Suggestions     []*SuggestAnswerItem `json:"suggestions"`
  PageNumber      int                  `json:"page_number"`
  TotalPagesCount int                  `json:"total_pages_count"`
  TotalItemsCount int                  `json:"total_items_count"`
}

type ProtoTransformer struct {
  ItemsMap map[*Item]int
  Items    []*stpb.Item
}

func NewProtoTransformer() *ProtoTransformer {
  return &ProtoTransformer{
    ItemsMap: make(map[*Item]int),
  }
}

func (pt *ProtoTransformer) TransformTrie(builder *SuggestTrieBuilder) (*stpb.SuggestTrie, error) {
  trie := &stpb.SuggestTrie{}
  for _, d := range builder.Descendants {
    descendant, err := pt.TransformTrie(d.Builder)
    if err != nil {
      return nil, err
    }
    trie.DescendantKeys = append(trie.DescendantKeys, uint32(d.Key))
    trie.DescendantTries = append(trie.DescendantTries, descendant)
  }
  for _, suggest := range builder.Suggest {
    trieItems := &stpb.ClassItems{
      Classes: suggest.Classes,
    }
    for _, item := range suggest.Suggest {
      if _, ok := pt.ItemsMap[item.OriginalItem]; !ok {
        dataStruct, err := structpb.NewStruct(item.OriginalItem.Data)
        if err != nil {
          return nil, err
        }
        pt.ItemsMap[item.OriginalItem] = len(pt.Items)
        pt.Items = append(pt.Items, &stpb.Item{
          Weight:       item.OriginalItem.Weight,
          OriginalText: item.OriginalItem.OriginalText,
          Data:         dataStruct,
        })
      }
      trieItems.ItemWeights = append(trieItems.ItemWeights, item.Weight)
      trieItems.ItemIndexes = append(trieItems.ItemIndexes, uint32(pt.ItemsMap[item.OriginalItem]))
    }
    trie.Items = append(trie.Items, trieItems)
  }
  return trie, nil
}

func Transform(builder *SuggestTrieBuilder) (*stpb.SuggestData, error) {
  pt := NewProtoTransformer()
  trie, err := pt.TransformTrie(builder)
  if err != nil {
    return nil, err
  }
  return &stpb.SuggestData{
    Trie:  trie,
    Items: pt.Items,
  }, nil
}

func BuildSuggest(items []*Item, maxItemsPerPrefix int, postfixWeightFactor float32) (*stpb.SuggestData, error) {
  veroheadItemsCount := maxItemsPerPrefix * 2
  builder := &SuggestTrieBuilder{}
  for idx, item := range items {
    itemClasses, err := extractItemClasses(item)
    if err != nil {
      return nil, fmt.Errorf("unable to extract item classes: %v", err)
    }
    builder.Add(0, item.NormalizedText, veroheadItemsCount, &SuggestTrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
      Classes:      itemClasses,
    })
    parts := strings.Split(item.NormalizedText, " ")
    for i := 1; i < len(parts); i++ {
      builder.Add(0, strings.Join(parts[i:], " "), veroheadItemsCount, &SuggestTrieItem{
        Weight:       item.Weight * postfixWeightFactor,
        OriginalItem: item,
        Classes:      itemClasses,
      })
    }
    if (idx+1)%100000 == 0 {
      log.Printf("addedd %d items of %d to suggest", idx+1, len(items))
    }
  }
  log.Printf("finalizing suggest")
  builder.Finalize(maxItemsPerPrefix)
  return Transform(builder)
}

func extractItemClasses(item *Item) ([]string, error) {
  b, err := json.Marshal(item.Data)
  if err != nil {
    return nil, fmt.Errorf("cannot convert data to json: %v", err)
  }
  params := &SuggestTrieItemClasses{}
  if err := json.Unmarshal(b, params); err != nil {
    return nil, fmt.Errorf("cannot parse json data: %v", err)
  }
  classes := params.Classes
  deprecatedClass := params.Class

  itemClassesMap := PrepareBoolMap(classes, false)
  if _, ok := itemClassesMap[deprecatedClass]; !ok {
    classes = append(classes, deprecatedClass)
  }
  return classes, nil
}
func doHighlight(originalPart string, originalSuggest string) []*SuggestionTextBlock {
  alphaLoweredPart := strings.ToLower(AlphaNormalizeString(originalPart))
  loweredSuggest := strings.ToLower(originalSuggest)

  partFields := strings.Fields(alphaLoweredPart)
  pos := 0
  var textBlocks []*SuggestionTextBlock
  for idx, partField := range partFields {
    suggestParts := strings.SplitN(loweredSuggest[pos:], partField, 2)
    if suggestParts[0] != "" {
      textBlocks = append(textBlocks, &SuggestionTextBlock{
        Text:      originalSuggest[pos : pos+len(suggestParts[0])],
        Highlight: false,
      })
    }
    textBlocks = append(textBlocks, &SuggestionTextBlock{
      Text:      originalSuggest[pos+len(suggestParts[0]) : pos+len(suggestParts[0])+len(partField)],
      Highlight: true,
    })
    if idx+1 == len(partFields) && len(suggestParts) == 2 && suggestParts[1] != "" {
      textBlocks = append(textBlocks, &SuggestionTextBlock{
        Text:      originalSuggest[pos+len(suggestParts[0])+len(partField) : pos+len(suggestParts[0])+len(partField)+len(suggestParts[1])],
        Highlight: false,
      })
    }
    pos += len(partField) + len(suggestParts[0])
  }
  return textBlocks
}

func GetSuggestItems(suggest *stpb.SuggestData, prefix []byte, classes, excludeClasses map[string]bool) []*stpb.Item {
  trie := suggest.Trie
  for _, c := range prefix {
    found := false
    for idx, key := range trie.DescendantKeys {
      if key != uint32(c) {
        continue
      }
      trie = trie.DescendantTries[idx]
      found = true
      break
    }
    if !found {
      return nil
    }
  }
  for len(trie.DescendantKeys) == 1 && len(trie.Items) == 0 {
    for _, d := range trie.DescendantTries {
      trie = d
      break
    }
  }
  var items []*stpb.Item
  seenItems := map[string]bool{}
  for _, suggestItems := range trie.Items {
    if !hasClass(suggestItems.Classes, classes) && len(classes) > 0 {
      continue
    }
    if hasClass(suggestItems.Classes, excludeClasses) {
      continue
    }
    for _, itemIdx := range suggestItems.ItemIndexes {
      item := suggest.Items[itemIdx]
      if _, ok := seenItems[item.OriginalText]; ok {
        continue
      }
      items = append(items, item)
      seenItems[item.OriginalText] = true
    }
  }

  sort.Slice(items, func(i, j int) bool {
    return items[i].Weight > items[j].Weight
  })
  return items
}

func hasClass(suggestClasses []string, requiredClasses map[string]bool) bool {
  for _, class := range suggestClasses {
    if _, ok := requiredClasses[strings.ToLower(class)]; ok {
      return true
    }
  }
  return false
}

func GetSuggest(suggest *stpb.SuggestData, originalPart string, normalizedPart string, classes, excludeClasses map[string]bool) []*SuggestAnswerItem {
  trieItems := GetSuggestItems(suggest, []byte(normalizedPart), classes, excludeClasses)
  items := make([]*SuggestAnswerItem, 0)
  if trieItems == nil {
    return items
  }
  for _, trieItem := range trieItems {
    items = append(items, &SuggestAnswerItem{
      Weight:     trieItem.Weight,
      Data:       trieItem.Data.AsMap(),
      TextBlocks: doHighlight(originalPart, trieItem.OriginalText),
    })
  }
  return items
}

func LoadSuggest(suggestDataPath string) (*stpb.SuggestData, error) {
  b, err := ioutil.ReadFile(suggestDataPath)
  if err != nil {
    return nil, err
  }
  suggestData := &stpb.SuggestData{}
  if err := proto.Unmarshal(b, suggestData); err != nil {
    return nil, err
  }
  return suggestData, nil
}
