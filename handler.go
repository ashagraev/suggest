package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/microcosm-cc/bluemonday"
  "log"
  "net/http"
)

type Handler struct {
  Suggest *SuggestData
  Policy  *bluemonday.Policy
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

func (h *Handler) HandleSuggestRequest(w http.ResponseWriter, r *http.Request) {
  writeCORSHeaders(w)
  part := r.URL.Query().Get("part")
  part = NormalizeString(part, h.Policy)
  items := h.Suggest.Get(part)
  reportSuccessData(w, items)
}
