package suggest

import (
  "container/heap"
  "reflect"
  "sort"
  "strings"
)

type SuggestTrieItem struct {
  Weight       float32
  OriginalItem *Item
}

type SuggestTrieDescendant struct {
  Key     byte
  Builder *SuggestTrieBuilder
}

type SuggestItems struct {
  Class   string
  Suggest []*SuggestTrieItem
}

func (s *SuggestItems) Len() int {
  return len(s.Suggest)
}

func (s *SuggestItems) Less(i, j int) bool {
  return s.Suggest[i].Weight < s.Suggest[j].Weight
}

func (s *SuggestItems) Swap(i, j int) {
  s.Suggest[i], s.Suggest[j] = s.Suggest[j], s.Suggest[i]
}

func (s *SuggestItems) Push(x interface{}) {
  s.Suggest = append(s.Suggest, x.(*SuggestTrieItem))
}

func (s *SuggestItems) Pop() interface{} {
  lastItem := s.Suggest[len(s.Suggest)-1]
  s.Suggest[len(s.Suggest)-1] = nil
  s.Suggest = s.Suggest[:len(s.Suggest)-1]
  return lastItem
}

func (s *SuggestItems) DeduplicateSuggest() {
  seenGroups := map[string]bool{}
  var deduplicatedItems []*SuggestTrieItem
  for _, item := range s.Suggest {
    group, ok := item.OriginalItem.Data["group"]
    if !ok {
      deduplicatedItems = append(deduplicatedItems, item)
      continue
    }
    if _, ok := seenGroups[group.(string)]; ok {
      continue
    }
    seenGroups[group.(string)] = true
    deduplicatedItems = append(deduplicatedItems, item)
  }
  s.Suggest = nil
  s.Suggest = deduplicatedItems
}

type SuggestTrieBuilder struct {
  Descendants []*SuggestTrieDescendant
  Suggest     []*SuggestItems
}

func (s *SuggestTrieBuilder) addItem(maxItemsPerPrefix int, item *SuggestTrieItem) {
  class := ""
  if knownClass, ok := item.OriginalItem.Data["class"]; ok {
    class = strings.ToLower(knownClass.(string))
  }
  for _, suggest := range s.Suggest {
    if suggest.Class == class {
      heap.Push(suggest, item)
      for len(suggest.Suggest) > maxItemsPerPrefix {
        heap.Pop(suggest)
      }
      return
    }
  }
  s.Suggest = append(s.Suggest, &SuggestItems{
    Class:   class,
    Suggest: []*SuggestTrieItem{item},
  })
}

func (s *SuggestTrieBuilder) Add(position int, text string, maxItemsPerPrefix int, item *SuggestTrieItem) {
  s.addItem(maxItemsPerPrefix, item)
  if position == len(text) {
    return
  }
  c := text[position]
  var descendant *SuggestTrieDescendant
  for _, d := range s.Descendants {
    if d.Key != c {
      continue
    }
    descendant = d
  }
  if descendant == nil {
    descendant = &SuggestTrieDescendant{
      Key:     c,
      Builder: &SuggestTrieBuilder{},
    }
    s.Descendants = append(s.Descendants, descendant)
  }
  descendant.Builder.Add(position+1, text, maxItemsPerPrefix, item)
}

func (s *SuggestTrieBuilder) Finalize(maxItemsPerPrefix int) {
  for _, descendant := range s.Descendants {
    if len(s.Descendants) == 1 && reflect.DeepEqual(descendant.Builder.Suggest, s.Suggest) {
      s.Suggest = nil
    }
  }
  for _, suggest := range s.Suggest {
    sort.Slice(suggest.Suggest, func(i, j int) bool {
      return suggest.Suggest[i].Weight > suggest.Suggest[j].Weight
    })
    suggest.DeduplicateSuggest()
    if len(suggest.Suggest) > maxItemsPerPrefix {
      suggest.Suggest = suggest.Suggest[:maxItemsPerPrefix]
    }
  }
  for _, descendant := range s.Descendants {
    descendant.Builder.Finalize(maxItemsPerPrefix)
  }
}
