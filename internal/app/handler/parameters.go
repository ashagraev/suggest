package handler

import (
	"encoding/json"
	"fmt"
	"main/internal/pkg/suggest"
	"main/pkg/utils"
	suggestRequestPb "main/proto/suggest/suggest_request"
	"math"
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

func (h *Handler) getPagingParameters(request *suggestRequestPb.Request) (*PagingParameters, error) {
	pagingParameters := &PagingParameters{}
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, pagingParameters)
	if err != nil {
		return nil, err
	}
	err = validatePagingParameters(pagingParameters)
	if err != nil {
		return nil, err
	}
	return pagingParameters, nil
}

func (pp *PagingParameters) Apply(suggestions suggest.Suggestions) *suggest.PaginatedSuggestResponse {
	itemsCount := len(suggestions)
	pagesCount := 1
	if pp.Count != 0 {
		pagesCount = int(math.Ceil(float64(len(suggestions)) / float64(pp.Count)))
	}

	offsetValue := pp.Page * pp.Count
	if offsetValue < itemsCount {
		suggestions = suggestions[offsetValue:]
	} else {
		suggestions = suggest.Suggestions{}
	}

	fetchValue := pp.Count
	if fetchValue < len(suggestions) {
		suggestions = suggestions[:fetchValue]
	}

	return &suggest.PaginatedSuggestResponse{
		Suggestions:     suggestions,
		PageNumber:      pp.Page,
		TotalPagesCount: pagesCount,
		TotalItemsCount: itemsCount,
	}
}

func (h *Handler) getSuggestionParameters(request *suggestRequestPb.Request) *suggest.SuggestionParameters {
	params := &suggest.SuggestionParameters{
		OriginalPart: request.Part,
	}
	if h.EqualShapedNormalize {
		params.NormalizedPart = utils.EqualShapedNormalizeString(request.Part, h.Policy)
	} else {
		params.NormalizedPart = utils.NormalizeString(request.Part, h.Policy)
	}
	params.Classes = utils.PrepareBoolMap(request.Class)
	params.ExcludeClasses = utils.PrepareBoolMap(request.ExcludeClass)
	return params
}
