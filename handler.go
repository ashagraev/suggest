package main

import (
  "github.com/microcosm-cc/bluemonday"
  "main/network"
  stpb "main/proto/suggest/suggest_trie"
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

func (h *Handler) HandleSuggestRequest(w http.ResponseWriter, r *http.Request) {
  network.WriteCORSHeaders(w)
  part := r.URL.Query().Get("part")
  if h.EqualShapedNormalize {
    part = ToEqualShapedLatin(part)
  }
  normalizedPart := part
  if h.EqualShapedNormalize {
    normalizedPart = EqualShapedNormalizeString(part, h.Policy)
  } else {
    normalizedPart = NormalizeString(part, h.Policy)
  }
  withVersion, _ := strconv.ParseBool(r.URL.Query().Get("with-version"))
  classes := r.URL.Query()["class"]
  classesMap := PrepareCheckMap(classes)
  excludeClasses := r.URL.Query()["exclude-class"]
  excludeClassesMap := PrepareCheckMap(excludeClasses)
  suggestions := GetSuggest(h.Suggest, part, normalizedPart, classesMap, excludeClassesMap)
  pagingParameters := NewPagingParameters(r.URL.Query())

  var resp interface{}
  if pagingParameters.PaginationOn {
    resp = pagingParameters.Apply(suggestions)
  } else {
    count := pagingParameters.Count
    if count != 0 && len(suggestions) > count {
      suggestions = suggestions[:count]
    }
    resp = suggestions
  }

  if !withVersion {
    network.ReportSuccessData(w, resp)
    return
  }

  switch response := resp.(type) {
  case *PaginatedSuggestResponse:
    response.Version = h.Suggest.Version
    network.ReportSuccessData(w, response)
  case []*SuggestAnswerItem:
    defaultResp := &SuggestResponse{
      Suggestions: response,
      Version:     h.Suggest.Version,
    }
    network.ReportSuccessData(w, defaultResp.Suggestions)
  }
}
