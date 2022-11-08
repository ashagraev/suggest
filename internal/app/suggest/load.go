package suggest

import (
  stpb "main/proto"
  "os"
  "google.golang.org/protobuf/proto"
)

func LoadSuggest(suggestDataPath string) (*stpb.SuggestData, error) {
  b, err := os.ReadFile(suggestDataPath)
  if err != nil {
    return nil, err
  }
  suggestData := &stpb.SuggestData{}
  if err := proto.Unmarshal(b, suggestData); err != nil {
    return nil, err
  }
  return suggestData, nil
}
