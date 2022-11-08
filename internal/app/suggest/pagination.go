package suggest

import (
  "fmt"
  "encoding/json"
  "math"
  suggestRequestPb "main/proto"
)

type PagingParameters struct {
  Count int `json:"count"`
  Page  int `json:"page"`
}

func validatePagingParameters(pp *PagingParameters) error {
  errTmpl := "parameter '%s' must be non-negative integer value"
  if pp.Page < 0 {
    return fmt.Errorf(errTmpl, "page")
  }
  if pp.Count < 0 {
    return fmt.Errorf(errTmpl, "count")
  }
  return nil
}

func GetPagingParameters(request *suggestRequestPb.Request) (*PagingParameters, error) {
  pp := &PagingParameters{}
  b, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }
  err = json.Unmarshal(b, pp)
  if err != nil {
    return nil, err
  }
  err = validatePagingParameters(pp)
  if err != nil {
    return nil, err
  }
  return pp, nil
}

func (s *Response) WithPagination(pp *PagingParameters) *Response {
  if pp.Page == 0 && pp.Count == 0 {
    return s
  }
  suggestions := s.Suggestions
  itemsCount := len(suggestions)

  pagesCount := 1
  if pp.Count != 0 {
    pagesCount = int(math.Ceil(float64(len(suggestions)) / float64(pp.Count)))
  }
  offsetValue := pp.Page * pp.Count
  if offsetValue < itemsCount {
    suggestions = suggestions[offsetValue:]
  } else {
    suggestions = []*SuggestionItem{}
  }
  fetchValue := pp.Count
  if fetchValue < len(suggestions) {
    suggestions = suggestions[:fetchValue]
  }

  return &Response{
    Suggestions: suggestions,
    Pagination: &Pagination{
      PageNumber:      pp.Page,
      TotalPagesCount: pagesCount,
      TotalItemsCount: itemsCount,
    },
  }
}
