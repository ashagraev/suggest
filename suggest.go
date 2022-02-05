package main

import (
  "log"
  "strings"
)

type SuggestData struct {
  Root  *SuggestTrie
  Items []*Item
}

type SuggestionTextBlock struct {
  Text      string `json:"text"`
  Highlight bool   `json:"hl"`
}

type SuggestAnswerItem struct {
  Weight     float32                `json:"weight"`
  Data       map[string]interface{} `json:"data"`
  TextBlocks []*SuggestionTextBlock `json:"text"`
}

func BuildSuggest(items []*Item, maxItemsPerPrefix int, postfixWeightFactor float32) *SuggestData {
  suggest := &SuggestData{
    Root:  NewSuggestionsTrie(),
    Items: items,
  }
  for idx, item := range items {
    suggest.Root.Add(0, item.NormalizedText, maxItemsPerPrefix, &SuggestTrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
    })
    parts := strings.Split(item.NormalizedText, " ")
    for i := 1; i < len(parts); i++ {
      suggest.Root.Add(0, strings.Join(parts[i:], " "), maxItemsPerPrefix, &SuggestTrieItem{
        Weight:       item.Weight * postfixWeightFactor,
        OriginalItem: item,
      })
    }
    if (idx+1)%100000 == 0 {
      log.Printf("addedd %d items of %d to suggest", idx+1, len(items))
    }
  }
  log.Printf("finalizing suggest")
  suggest.Root.Finalize()
  return suggest
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

func (sd *SuggestData) Get(originalPart string, normalizedPart string) []*SuggestAnswerItem {
  trieItems := sd.Root.Get([]byte(normalizedPart))
  items := make([]*SuggestAnswerItem, 0)
  if trieItems == nil {
    return items
  }
  for _, trieItem := range trieItems.Items {
    items = append(items, &SuggestAnswerItem{
      Weight:     trieItem.Weight,
      Data:       trieItem.OriginalItem.Data,
      TextBlocks: doHighlight(originalPart, trieItem.OriginalItem.OriginalText),
    })
  }
  return items
}
