package suggest

import (
  "google.golang.org/protobuf/types/known/structpb"
  "main/proto"
)

type ProtoTransformer struct {
  ItemsMap map[*Item]int
  Items    []*proto.Item
}

func newProtoTransformer() *ProtoTransformer {
  return &ProtoTransformer{
    ItemsMap: make(map[*Item]int),
  }
}

func (pt *ProtoTransformer) transformTrie(builder *TrieBuilder) (*proto.SuggestTrie, error) {
  trie := &proto.SuggestTrie{}
  for _, d := range builder.Descendants {
    descendant, err := pt.transformTrie(d.Builder)
    if err != nil {
      return nil, err
    }
    trie.DescendantKeys = append(trie.DescendantKeys, uint32(d.Key))
    trie.DescendantTries = append(trie.DescendantTries, descendant)
  }
  for _, suggest := range builder.Suggest {
    trieItems := &proto.ClassesItems{
      Classes: suggest.Classes,
    }
    for _, item := range suggest.Suggest {
      if _, ok := pt.ItemsMap[item.OriginalItem]; !ok {
        dataStruct, err := structpb.NewStruct(item.OriginalItem.Data)
        if err != nil {
          return nil, err
        }
        pt.ItemsMap[item.OriginalItem] = len(pt.Items)
        pt.Items = append(pt.Items, &proto.Item{
          Weight:       item.OriginalItem.Weight,
          OriginalText: item.OriginalItem.OriginalText,
          Data:         dataStruct,
        })
      }
      trieItems.ItemWeights = append(trieItems.ItemWeights, item.Weight)
      trieItems.ItemIndexes = append(trieItems.ItemIndexes, uint32(pt.ItemsMap[item.OriginalItem]))
    }
    trie.Items = append(trie.Items, trieItems)
  }
  return trie, nil
}

func transform(builder *TrieBuilder) (*proto.SuggestData, error) {
  pt := newProtoTransformer()
  trie, err := pt.transformTrie(builder)
  if err != nil {
    return nil, err
  }
  return &proto.SuggestData{
    Trie:  trie,
    Items: pt.Items,
  }, nil
}
