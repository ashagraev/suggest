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

type SuggestTrie struct {
  Descendants map[byte]*SuggestTrie
  Suggest     *SuggestTrieItems
}

func NewSuggestionsTrie() *SuggestTrie {
  return &SuggestTrie{
    Descendants: make(map[byte]*SuggestTrie),
    Suggest:     &SuggestTrieItems{},
  }
}

func (st *SuggestTrie) Add(position int, text string, maxItemsPerPrefix int, item *SuggestTrieItem) {
  heap.Push(st.Suggest, item)
  for st.Suggest.Len() > maxItemsPerPrefix {
    heap.Pop(st.Suggest)
  }
  if position == len(text) {
    return
  }
  c := text[position]
  if _, ok := st.Descendants[c]; !ok {
    st.Descendants[c] = NewSuggestionsTrie()
  }
  st.Descendants[c].Add(position+1, text, maxItemsPerPrefix, item)
}

func (st *SuggestTrie) Get(prefix []byte) *SuggestTrieItems {
  trie := st
  for _, c := range prefix {
    d, ok := trie.Descendants[c]
    if !ok {
      return nil
    }
    trie = d
  }
  for len(trie.Descendants) == 1 && trie.Suggest.Len() == 0 {
    for _, d := range trie.Descendants {
      trie = d
      break
    }
  }
  return trie.Suggest
}

func (st *SuggestTrie) Finalize(maxItemsPerPrefix int) {
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
    if len(st.Descendants) == 1 && reflect.DeepEqual(descendant.Suggest, st.Suggest) {
      st.Suggest.Items = nil
    }
    descendant.Finalize(maxItemsPerPrefix)
  }
}
