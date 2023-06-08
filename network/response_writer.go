package network

import (
  "bytes"
  "encoding/json"
  "fmt"
  "log"
  "net/http"
)

func WriteCORSHeaders(w http.ResponseWriter) {
  w.Header().Add("Access-Control-Allow-Origin", "*")
  w.Header().Add("Access-Control-Allow-Methods", "OPTIONS,POST,GET")
}

func ReportServerError(w http.ResponseWriter, message string) {
  WriteCORSHeaders(w)
  w.WriteHeader(http.StatusInternalServerError)
  if _, err := w.Write([]byte(message)); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}

func ReportSuccessMessage(w http.ResponseWriter, message string) {
  WriteCORSHeaders(w)
  w.WriteHeader(http.StatusOK)
  if _, err := w.Write([]byte(message)); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}

func ReportSuccessData(w http.ResponseWriter, data interface{}) {
  j, err := json.Marshal(data)
  if err != nil {
    ReportServerError(w, fmt.Sprintf("%v", err))
    return
  }
  var b bytes.Buffer
  if err := json.Indent(&b, j, "", "  "); err != nil {
    ReportServerError(w, fmt.Sprintf("%v", err))
    return
  }
  WriteCORSHeaders(w)
  w.WriteHeader(http.StatusOK)
  if _, err := w.Write(b.Bytes()); err != nil {
    log.Printf("cannot write a message: %v", err)
  }
}
