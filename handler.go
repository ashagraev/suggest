package main

import (
  "github.com/microcosm-cc/bluemonday"
  "main/network"
  stpb "main/proto/suggest/suggest_trie"
  "main/tools"
  "math"
  "net/http"
  "net/url"
  "strconv"
)

type Handler struct {
  Suggest              *stpb.SuggestData
  Policy               *bluemonday.Policy
  EqualShapedNormalize bool
}

func (h *Handler) HandleHealthRequest(w http.ResponseWriter, _ *http.Request) {
  network.ReportSuccessMessage(w, "OK")
}

type PagingParameters struct {
  Count        int
  Page         int
  PaginationOn bool
}

func NewPagingParameters(query url.Values) *PagingParameters {
  pagingParameters := &PagingParameters{}
  if count, err := strconv.ParseInt(query.Get("count"), 10, 64); err == nil { // no err
    pagingParameters.Count = int(count)
  }
  if page, err := strconv.ParseInt(query.Get("page"), 10, 64); err == nil { // no err
    pagingParameters.Page = int(page)
    pagingParameters.PaginationOn = true
  }
  return pagingParameters
}

func (pp *PagingParameters) Apply(suggestions []*SuggestAnswerItem) *PaginatedSuggestResponse {
  pagesCount := 1
  if pp.Count != 0 {
    pagesCount = int(math.Ceil(float64(len(suggestions)) / float64(pp.Count)))
  }
  itemsCount := len(suggestions)
  if pp.Page != 0 && pp.Count != 0 {
    skip := pp.Page * pp.Count
    if len(suggestions) > skip {
      suggestions = suggestions[skip:]
    } else {
      suggestions = nil
    }
  }
  if pp.Count != 0 && len(suggestions) > pp.Count {
    suggestions = suggestions[:pp.Count]
  }
  return &PaginatedSuggestResponse{
    Suggestions:     suggestions,
    PageNumber:      pp.Page,
    TotalPagesCount: pagesCount,
    TotalItemsCount: itemsCount,
  }
}

func generateResponse(
  suggestions []*SuggestAnswerItem,
  pagingParameters *PagingParameters,
) interface{} {

  if pagingParameters.PaginationOn {
    response := pagingParameters.Apply(suggestions)
    return response
  }

  count := pagingParameters.Count
  if count != 0 && len(suggestions) > count {
    suggestions = suggestions[:count]
  }

  return suggestions
}

func (h *Handler) HandleSuggestRequest(w http.ResponseWriter, r *http.Request) {
  network.WriteCORSHeaders(w)
  part := r.URL.Query().Get("part")
  if h.EqualShapedNormalize {
    part = tools.ToEqualShapedLatin(part)
  }
  normalizedPart := part
  if h.EqualShapedNormalize {
    normalizedPart = tools.EqualShapedNormalizeString(part, h.Policy)
  } else {
    normalizedPart = tools.NormalizeString(part, h.Policy)
  }
  classes := r.URL.Query()["class"]
  classesMap := tools.PrepareCheckMap(classes)
  excludeClasses := r.URL.Query()["exclude-class"]
  excludeClassesMap := tools.PrepareCheckMap(excludeClasses)
  suggestions := GetSuggest(h.Suggest, part, normalizedPart, classesMap, excludeClassesMap)
  pagingParameters := NewPagingParameters(r.URL.Query())

  network.ReportSuccessData(w, generateResponse(suggestions, pagingParameters))
}
