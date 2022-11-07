package suggest

import (
	"container/heap"
	"reflect"
	"sort"
	"strings"
)

type TrieItem struct {
	Weight       float32
	OriginalItem *Item
}

type TrieDescendant struct {
	Key     byte
	Builder *TrieBuilder
}

type ItemsHeap struct {
	Class   string
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
	s.Suggest = append(s.Suggest, x.(*TrieItem))
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
	s.Suggest = append(s.Suggest, &ItemsHeap{
		Class:   class,
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
