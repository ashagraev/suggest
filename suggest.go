package main

import (
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/types/known/structpb"
  "log"
  stpb "main/proto/suggest/suggest_trie"
  "os"
  "sort"
  "strings"
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
      Class: suggest.Class,
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

func BuildSuggest(
  items []*Item,
  maxItemsPerPrefix int,
  postfixWeightFactor float32,
  disableNormalizedParts bool,
) (*stpb.SuggestData, error) {
  overheadItemsCount := maxItemsPerPrefix * 2
  builder := &SuggestTrieBuilder{}
  for idx, item := range items {
    builder.Add(0, item.NormalizedText, overheadItemsCount, &SuggestTrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
    })

    if !disableNormalizedParts {
      parts := strings.Split(item.NormalizedText, " ")
      for i := 1; i < len(parts); i++ {
        builder.Add(0, strings.Join(parts[i:], " "), overheadItemsCount, &SuggestTrieItem{
          Weight:       item.Weight * postfixWeightFactor,
          OriginalItem: item,
        })
      }
    }

    if (idx+1)%100000 == 0 {
      log.Printf("added %d items of %d to suggest", idx+1, len(items))
    }
  }
  log.Printf("finalizing suggest")
  builder.Finalize(maxItemsPerPrefix)
  return Transform(builder)
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
  for _, suggestItems := range trie.Items {
    for _, class := range suggestItems.Classes {
      if _, ok := excludeClasses[class]; ok {
        continue
      }
      if _, ok := classes[class]; !ok && len(classes) > 0 {
        continue
      }
    }
    if _, ok := classes[suggestItems.Class]; !ok && len(classes) > 0 {
      continue
    }
    for _, itemIdx := range suggestItems.ItemIndexes {
      items = append(items, suggest.Items[itemIdx])
    }
  }
  sort.Slice(items, func(i, j int) bool {
    return items[i].Weight > items[j].Weight
  })
  return items
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
  b, err := os.ReadFile(suggestDataPath)
  if err != nil {
    return nil, err
  }
  suggestData := &stpb.SuggestData{}
  if err := proto.Unmarshal(b, suggestData); err != nil {
    return nil, err
  }
  return suggestData, nil
}
