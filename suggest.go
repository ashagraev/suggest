package main

import (
  "log"
  "strings"
)

type SuggestData struct {
  Root  *SuggestTrie
  Items []*Item
}

func BuildSuggest(items []*Item, maxItemsPerPrefix int, postfixWeightFactor float32) *SuggestData {
  suggest := &SuggestData{
    Root:  NewSuggestionsTrie(),
    Items: items,
  }
  for idx, item := range items {
    suggest.Root.Add(0, item.Text, maxItemsPerPrefix, &SuggestTrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
    })
    parts := strings.Split(item.Text, " ")
    for i := 1; i < len(parts); i++ {
      suggest.Root.Add(0, strings.Join(parts[i:], " "), maxItemsPerPrefix, &SuggestTrieItem{
        Weight:       item.Weight * postfixWeightFactor,
        OriginalItem: item,
      })
    }
    if idx%100000 == 0 {
      log.Printf("addedd %d items of %d to suggest", idx, len(items))
    }
  }
  log.Printf("finalizing suggest")
  suggest.Root.Finalize()
  return suggest
}

func (sd *SuggestData) Get(part string) []*Item {
  trieItems := sd.Root.Get([]byte(part))
  var items []*Item
  for _, trieItem := range trieItems.Items {
    item := *trieItem.OriginalItem
    item.Weight = trieItem.Weight
    items = append(items, &item)
  }
  return items
}
