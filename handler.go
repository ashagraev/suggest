package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/microcosm-cc/bluemonday"
  "log"
  stpb "main/proto/suggest/suggest_trie"
  "math"
  "net/http"
  "net/url"
  "strconv"
  "strings"
)

type Handler struct {
  Suggest              *stpb.SuggestData
  Policy               *bluemonday.Policy
  EqualShapedNormalize bool
}

func writeCORSHeaders(w http.ResponseWriter) {
  w.Header().Add("Access-Control-Allow-Origin", "*")
  w.Header().Add("Access-Control-Allow-Methods", "OPTIONS,POST,GET")
}

func reportServerError(w http.ResponseWriter, message string) {
  writeCORSHeaders(w)
  w.WriteHeader(http.StatusInternalServerError)
  if _, err := w.Write([]byte(message)); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}

func reportSuccessMessage(w http.ResponseWriter, message string) {
  writeCORSHeaders(w)
  w.WriteHeader(http.StatusOK)
  if _, err := w.Write([]byte(message)); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}

func reportSuccessData(w http.ResponseWriter, data interface{}) {
  j, err := json.Marshal(data)
  if err != nil {
    reportServerError(w, fmt.Sprintf("%v", err))
    return
  }
  var b bytes.Buffer
  if err := json.Indent(&b, j, "", "  "); err != nil {
    reportServerError(w, fmt.Sprintf("%v", err))
    return
  }
  writeCORSHeaders(w)
  w.WriteHeader(http.StatusOK)
  if _, err := w.Write(b.Bytes()); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}

func (h *Handler) HandleHealthRequest(w http.ResponseWriter, _ *http.Request) {
  reportSuccessMessage(w, "OK")
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
  writeCORSHeaders(w)
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
  classes := r.URL.Query()["class"]
  classesMap := map[string]bool{}
  for _, class := range classes {
    if class != "" {
      classesMap[strings.ToLower(class)] = true
    }
  }
  suggestions := GetSuggest(h.Suggest, part, normalizedPart, classesMap)
  pagingParameters := NewPagingParameters(r.URL.Query())
  if pagingParameters.PaginationOn {
    reportSuccessData(w, pagingParameters.Apply(suggestions))
  } else {
    if count, err := strconv.ParseInt(r.URL.Query().Get("count"), 10, 64); err == nil { // no err
      if count != 0 && len(suggestions) > int(count) {
        suggestions = suggestions[:count]
      }
    }
    reportSuccessData(w, suggestions)
  }
}
