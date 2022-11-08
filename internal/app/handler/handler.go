package handler

import (
  "fmt"
  "net/http"
  "github.com/microcosm-cc/bluemonday"
  "main/pkg/utils"
  "main/internal/app/suggest"
  "main/proto"
)

type Handler struct {
  Suggest              *proto.SuggestData
  Policy               *bluemonday.Policy
  EqualShapedNormalize bool
}

func (h *Handler) formSuggestPayload(request *proto.Request) *suggest.Payload {
  params := &suggest.Payload{
    OriginalPart: request.Part,
  }
  if h.EqualShapedNormalize {
    params.NormalizedPart = utils.EqualShapedNormalizeString(request.Part, h.Policy)
  } else {
    params.NormalizedPart = utils.NormalizeString(request.Part, h.Policy)
  }
  params.Classes = utils.PrepareBoolMap(request.Class, true)
  params.ExcludeClasses = utils.PrepareBoolMap(request.ExcludeClass, true)
  return params
}

func (h *Handler) HandleSuggestRequest(w http.ResponseWriter, r *http.Request) {
  if r.Method == http.MethodOptions {
    utils.WriteCORSHeaders(w)
    return
  }
  request := &proto.Request{}
  if err := utils.ReadRequest(r, request); err != nil {
    utils.ReportServerError(w, fmt.Sprintf("cannot read request: %v", err))
    return
  }
  pagingParams, err := suggest.GetPagingParameters(request)
  if err != nil {
    utils.ReportRequestError(w, fmt.Sprintf("bad request: %v", err))
    return
  }
  payload := h.formSuggestPayload(request)
  suggestions := suggest.GetSuggest(h.Suggest, payload).WithPagination(pagingParams)

  utils.ReportSuccessData(w, suggestions)
}

func (h *Handler) HandleHealthRequest(w http.ResponseWriter, _ *http.Request) {
  utils.ReportSuccessMessage(w, "OK")
}
