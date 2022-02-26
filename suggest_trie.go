package main

import (
  "container/heap"
  "reflect"
  "sort"
)

type SuggestTrieItem struct {
  Weight       float32
  OriginalItem *Item
}

type SuggestTrieDescendant struct {
  Key     byte
  Builder *SuggestTrieBuilder
}

type SuggestTrieBuilder struct {
  Descendants []*SuggestTrieDescendant
  Suggest     []*SuggestTrieItem
}

func (s *SuggestTrieBuilder) Len() int {
  return len(s.Suggest)
}

func (s *SuggestTrieBuilder) Less(i, j int) bool {
  return s.Suggest[i].Weight < s.Suggest[j].Weight
}

func (s *SuggestTrieBuilder) Swap(i, j int) {
  s.Suggest[i], s.Suggest[j] = s.Suggest[j], s.Suggest[i]
}

func (s *SuggestTrieBuilder) Push(x interface{}) {
  s.Suggest = append(s.Suggest, x.(*SuggestTrieItem))
}

func (s *SuggestTrieBuilder) Pop() interface{} {
  lastItem := s.Suggest[len(s.Suggest)-1]
  s.Suggest[len(s.Suggest)-1] = nil
  s.Suggest = s.Suggest[:len(s.Suggest)-1]
  return lastItem
}

func (s *SuggestTrieBuilder) Add(position int, text string, maxItemsPerPrefix int, item *SuggestTrieItem) {
  heap.Push(s, item)
  for len(s.Suggest) > maxItemsPerPrefix {
    heap.Pop(s)
  }
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

func (s *SuggestTrieBuilder) DeduplicateSuggest() {
  var deduplicatedItems []*SuggestTrieItem
  seenGroups := map[string]bool{}
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

func (s *SuggestTrieBuilder) Finalize(maxItemsPerPrefix int) {
  for _, descendant := range s.Descendants {
    if len(s.Descendants) == 1 && reflect.DeepEqual(descendant.Builder.Suggest, s.Suggest) {
      s.Suggest = nil
    }
  }
  sort.Slice(s.Suggest, func(i, j int) bool {
    return s.Suggest[i].Weight > s.Suggest[j].Weight
  })
  s.DeduplicateSuggest()
  if len(s.Suggest) > maxItemsPerPrefix {
    s.Suggest = s.Suggest[:maxItemsPerPrefix]
  }
  for _, descendant := range s.Descendants {
    descendant.Builder.Finalize(maxItemsPerPrefix)
  }
}
