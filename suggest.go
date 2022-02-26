package main

import (
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/types/known/structpb"
  "io/ioutil"
  "log"
  stpb "main/proto/suggest/suggest_trie"
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
  for k, v := range builder.Descendants {
    descendant, err := pt.TransformTrie(v)
    if err != nil {
      return nil, err
    }
    trie.Descendants = append(trie.Descendants, &stpb.Descendant{
      Key:  uint32(k),
      Trie: descendant,
    })
  }
  for _, item := range builder.Suggest.Items {
    if _, ok := pt.ItemsMap[item.OriginalItem]; !ok {
      dataStruct, err := structpb.NewStruct(item.OriginalItem.Data)
      if err != nil {
        return nil, err
      }
      pt.ItemsMap[item.OriginalItem] = len(pt.Items)
      pt.Items = append(pt.Items, &stpb.Item{
        Weight:         item.OriginalItem.Weight,
        OriginalText:   item.OriginalItem.OriginalText,
        NormalizedText: item.OriginalItem.NormalizedText,
        Data:           dataStruct,
      })
    }
    trie.Items = append(trie.Items, &stpb.SuggestTrieItem{
      Weight:          item.Weight,
      OriginalItemIdx: uint32(pt.ItemsMap[item.OriginalItem]),
    })
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
  builder := NewSuggestionsTrieBuilder()
  for idx, item := range items {
    builder.Add(0, item.NormalizedText, maxItemsPerPrefix*5, &SuggestTrieItem{
      Weight:       item.Weight,
      OriginalItem: item,
    })
    parts := strings.Split(item.NormalizedText, " ")
    for i := 1; i < len(parts); i++ {
      builder.Add(0, strings.Join(parts[i:], " "), maxItemsPerPrefix*5, &SuggestTrieItem{
        Weight:       item.Weight * postfixWeightFactor,
        OriginalItem: item,
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

func GetSuggestItems(suggest *stpb.SuggestData, prefix []byte) []*stpb.SuggestTrieItem {
  trie := suggest.Trie
  for _, c := range prefix {
    found := false
    for _, d := range trie.Descendants {
      if d.Key != uint32(c) {
        continue
      }
      trie = d.Trie
      found = true
      break
    }
    if !found {
      return nil
    }
  }
  for len(trie.Descendants) == 1 && len(trie.Items) == 0 {
    for _, d := range trie.Descendants {
      trie = d.Trie
      break
    }
  }
  return trie.Items
}

func GetSuggest(suggest *stpb.SuggestData, originalPart string, normalizedPart string) []*SuggestAnswerItem {
  trieItems := GetSuggestItems(suggest, []byte(normalizedPart))
  items := make([]*SuggestAnswerItem, 0)
  if trieItems == nil {
    return items
  }
  for _, trieItem := range trieItems {
    originalItem := suggest.Items[trieItem.OriginalItemIdx]
    items = append(items, &SuggestAnswerItem{
      Weight:     trieItem.Weight,
      Data:       originalItem.Data.AsMap(),
      TextBlocks: doHighlight(originalPart, originalItem.OriginalText),
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
