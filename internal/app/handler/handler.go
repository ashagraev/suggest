package handler

import (
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"main/internal/pkg/suggest"
	"main/pkg/utils"
	suggestRequestPb "main/proto/suggest/suggest_request"
	suggestTriePb "main/proto/suggest/suggest_trie"
	"net/http"
)

type Handler struct {
	Suggest              *suggestTriePb.SuggestData
	Policy               *bluemonday.Policy
	EqualShapedNormalize bool
}

func (h *Handler) HandleSuggestRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		utils.WriteCORSHeaders(w)
		return
	}
	request := &suggestRequestPb.Request{}
	if err := utils.ReadRequest(r, request); err != nil {
		utils.ReportServerError(w, fmt.Sprintf("cannot read request: %v", err))
		return
	}
	pagingParams, err := h.getPagingParameters(request)
	if err != nil {
		utils.ReportRequestError(w, fmt.Sprintf("bad request: %v", err))
		return
	}
	suggestionParams := h.getSuggestionParameters(request)
	suggestions := suggest.GetSuggest(h.Suggest, suggestionParams)

	paginatedSuggestions := pagingParams.Apply(suggestions)
	utils.ReportSuccessData(w, paginatedSuggestions)
}

func (h *Handler) HandleHealthRequest(w http.ResponseWriter, _ *http.Request) {
	utils.ReportSuccessMessage(w, "OK")
}
