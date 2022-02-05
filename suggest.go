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

func doHighlight(prefix string, suggest string) []*SuggestionTextBlock {
  suggestParts := strings.SplitN(suggest, prefix, 2)
  var textBlocks []*SuggestionTextBlock
  for idx := range suggestParts {
    if suggestParts[idx] != "" {
      textBlocks = append(textBlocks, &SuggestionTextBlock{
        Text:      suggestParts[idx],
        Highlight: false,
      })
    }
    if idx+1 != len(suggestParts) {
      textBlocks = append(textBlocks, &SuggestionTextBlock{
        Text:      prefix,
        Highlight: true,
      })
    }
  }
  return textBlocks
}

func (sd *SuggestData) Get(part string) []*SuggestAnswerItem {
  trieItems := sd.Root.Get([]byte(part))
  items := make([]*SuggestAnswerItem, 0)
  if trieItems == nil {
    return items
  }
  for _, trieItem := range trieItems.Items {
    items = append(items, &SuggestAnswerItem{
      Weight:     trieItem.Weight,
      Data:       trieItem.OriginalItem.Data,
      TextBlocks: doHighlight(part, trieItem.OriginalItem.NormalizedText),
    })
  }
  return items
}
