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

type SuggestTrieItems struct {
  Items []*SuggestTrieItem
}

func (s *SuggestTrieItems) Len() int {
  return len(s.Items)
}

func (s *SuggestTrieItems) Less(i, j int) bool {
  return s.Items[i].Weight < s.Items[j].Weight
}

func (s *SuggestTrieItems) Swap(i, j int) {
  s.Items[i], s.Items[j] = s.Items[j], s.Items[i]
}

func (s *SuggestTrieItems) Push(x interface{}) {
  s.Items = append(s.Items, x.(*SuggestTrieItem))
}

func (s *SuggestTrieItems) Pop() interface{} {
  lastItem := s.Items[len(s.Items)-1]
  s.Items[len(s.Items)-1] = nil
  s.Items = s.Items[:len(s.Items)-1]
  return lastItem
}

type SuggestTrieDescendant struct {
  Key     byte
  Builder *SuggestTrieBuilder
}

type SuggestTrieBuilder struct {
  Descendants []*SuggestTrieDescendant
  Suggest     *SuggestTrieItems
}

func NewSuggestionsTrieBuilder() *SuggestTrieBuilder {
  return &SuggestTrieBuilder{
    Suggest: &SuggestTrieItems{},
  }
}

func (st *SuggestTrieBuilder) Add(position int, text string, maxItemsPerPrefix int, item *SuggestTrieItem) {
  heap.Push(st.Suggest, item)
  for st.Suggest.Len() > maxItemsPerPrefix {
    heap.Pop(st.Suggest)
  }
  if position == len(text) {
    return
  }
  c := text[position]
  var descendant *SuggestTrieDescendant
  for _, d := range st.Descendants {
    if d.Key != c {
      continue
    }
    descendant = d
  }
  if descendant == nil {
    descendant = &SuggestTrieDescendant{
      Key:     c,
      Builder: NewSuggestionsTrieBuilder(),
    }
    st.Descendants = append(st.Descendants, descendant)
  }
  descendant.Builder.Add(position+1, text, maxItemsPerPrefix, item)
}

func (st *SuggestTrieBuilder) Finalize(maxItemsPerPrefix int) {
  for _, descendant := range st.Descendants {
    if len(st.Descendants) == 1 && reflect.DeepEqual(descendant.Builder.Suggest, &SuggestTrieItems{Items: st.Suggest.Items}) {
      st.Suggest.Items = nil
    }
  }
  sort.Slice(st.Suggest.Items, func(i, j int) bool {
    return st.Suggest.Items[i].Weight > st.Suggest.Items[j].Weight
  })
  var deduplicatedItems []*SuggestTrieItem
  seenGroups := map[string]bool{}
  for _, item := range st.Suggest.Items {
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
  st.Suggest.Items = deduplicatedItems
  if len(st.Suggest.Items) > maxItemsPerPrefix {
    st.Suggest.Items = st.Suggest.Items[:maxItemsPerPrefix]
  }
  for _, descendant := range st.Descendants {
    descendant.Builder.Finalize(maxItemsPerPrefix)
  }
}
