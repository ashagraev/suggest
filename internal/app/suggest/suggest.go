package suggest

import (
  "main/pkg/utils"
  "sort"
  "strings"
  stpb "main/proto"
)

func GetSuggest(suggest *stpb.SuggestData, params *Payload) *Response {
  trieItems := getSuggestItems(suggest, params)
  items := make([]*SuggestionItem, 0)
  for _, trieItem := range trieItems {
    items = append(items, &SuggestionItem{
      Weight:     trieItem.Weight,
      Data:       trieItem.Data.AsMap(),
      TextBlocks: doHighlight(params.OriginalPart, trieItem.OriginalText),
    })
  }
  return &Response{
    Suggestions: items,
  }
}

func getSuggestItems(suggest *stpb.SuggestData, params *Payload) []*stpb.Item {
  includeClasses := params.Classes
  excludeClasses := params.ExcludeClasses

  prefix := []byte(params.NormalizedPart)
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
    addItems := false
    for _, class := range suggestItems.Classes {
      class := strings.ToLower(class)
      if _, ok := excludeClasses[class]; ok {
        addItems = false
        break
      }
      if _, ok := includeClasses[class]; !ok && len(includeClasses) > 0 {
        continue
      }
      addItems = true
    }
    if !addItems {
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

func doHighlight(originalPart string, originalSuggest string) []*HighlightTextBlock {
  alphaLoweredPart := strings.ToLower(utils.AlphaNormalizeString(originalPart))
  loweredSuggest := strings.ToLower(originalSuggest)

  partFields := strings.Fields(alphaLoweredPart)
  pos := 0
  var textBlocks []*HighlightTextBlock
  for idx, partField := range partFields {
    suggestParts := strings.SplitN(loweredSuggest[pos:], partField, 2)
    if suggestParts[0] != "" {
      textBlocks = append(textBlocks, &HighlightTextBlock{
        Text:      originalSuggest[pos : pos+len(suggestParts[0])],
        Highlight: false,
      })
    }
    textBlocks = append(textBlocks, &HighlightTextBlock{
      Text:      originalSuggest[pos+len(suggestParts[0]) : pos+len(suggestParts[0])+len(partField)],
      Highlight: true,
    })
    if idx+1 == len(partFields) && len(suggestParts) == 2 && suggestParts[1] != "" {
      textBlocks = append(textBlocks, &HighlightTextBlock{
        Text:      originalSuggest[pos+len(suggestParts[0])+len(partField) : pos+len(suggestParts[0])+len(partField)+len(suggestParts[1])],
        Highlight: false,
      })
    }
    pos += len(partField) + len(suggestParts[0])
  }
  return textBlocks
}
