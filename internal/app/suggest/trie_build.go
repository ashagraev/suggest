package suggest

import (
  "reflect"
  "sort"
  "main/pkg/utils"
)

type TrieItem struct {
  Weight       float32
  OriginalItem *Item
  Classes      []string
}

type TrieItemClasses struct {
  Classes []string `json:"classes"`
  Class   string   `json:"class"` // deprecated
}

type TrieDescendant struct {
  Key     byte
  Builder *TrieBuilder
}

type ItemsHeap struct {
  Classes []string
  Suggest []*TrieItem
}

func (s *ItemsHeap) Len() int {
  return len(s.Suggest)
}

func (s *ItemsHeap) Less(i, j int) bool {
  return s.Suggest[i].Weight < s.Suggest[j].Weight
}

func (s *ItemsHeap) Swap(i, j int) {
  s.Suggest[i], s.Suggest[j] = s.Suggest[j], s.Suggest[i]
}

func (s *ItemsHeap) Push(x interface{}) {
  item := x.(*TrieItem)
  m := utils.PrepareBoolMap(s.Classes, false)
  for _, c := range item.Classes {
    if _, ok := m[c]; !ok {
      s.Classes = append(s.Classes, c)
    }
  }
  s.Suggest = append(s.Suggest, item)
}

func (s *ItemsHeap) Pop() interface{} {
  lastItem := s.Suggest[len(s.Suggest)-1]
  s.Suggest[len(s.Suggest)-1] = nil
  s.Suggest = s.Suggest[:len(s.Suggest)-1]
  return lastItem
}

func (s *ItemsHeap) DeduplicateSuggest() {
  seenGroups := map[string]bool{}
  var deduplicatedItems []*TrieItem
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

type TrieBuilder struct {
  Descendants []*TrieDescendant
  Suggest     []*ItemsHeap
}

func (s *TrieBuilder) addItem(maxItemsPerPrefix int, item *TrieItem) {
  itemClassesMap := utils.PrepareBoolMap(item.Classes, false)
  for _, suggest := range s.Suggest {
    for _, suggestClass := range suggest.Classes {
      _, ok := itemClassesMap[suggestClass]
      if !ok {
        continue
      }
      suggest.Push(item)
      for suggest.Len() > maxItemsPerPrefix {
        suggest.Pop()
      }
      return
    }
  }
  s.Suggest = append(s.Suggest, &ItemsHeap{
    Classes: item.Classes,
    Suggest: []*TrieItem{item},
  })
}

func (s *TrieBuilder) Add(position int, text string, maxItemsPerPrefix int, item *TrieItem) {
  s.addItem(maxItemsPerPrefix, item)
  if position == len(text) {
    return
  }
  c := text[position]
  var descendant *TrieDescendant
  for _, d := range s.Descendants {
    if d.Key != c {
      continue
    }
    descendant = d
  }
  if descendant == nil {
    descendant = &TrieDescendant{
      Key:     c,
      Builder: &TrieBuilder{},
    }
    s.Descendants = append(s.Descendants, descendant)
  }
  descendant.Builder.Add(position+1, text, maxItemsPerPrefix, item)
}

func (s *TrieBuilder) Finalize(maxItemsPerPrefix int) {
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
