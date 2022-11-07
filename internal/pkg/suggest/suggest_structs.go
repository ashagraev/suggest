package suggest

import stpb "main/proto/suggest/suggest_trie"

type Suggestions []*SuggestionAnswerItem

type SuggestionTextBlock struct {
	Text      string `json:"text"`
	Highlight bool   `json:"hl"`
}

type SuggestionAnswerItem struct {
	Weight     float32                `json:"weight"`
	Data       map[string]interface{} `json:"data"`
	TextBlocks []*SuggestionTextBlock `json:"text"`
}

type PaginatedSuggestResponse struct {
	Suggestions     Suggestions `json:"suggestions"`
	PageNumber      int         `json:"page_number"`
	TotalPagesCount int         `json:"total_pages_count"`
	TotalItemsCount int         `json:"total_items_count"`
}

type ProtoTransformer struct {
	ItemsMap map[*Item]int
	Items    []*stpb.Item
}

type SuggestionParameters struct {
	OriginalPart   string
	NormalizedPart string
	Classes        map[string]bool
	ExcludeClasses map[string]bool
	Display        int
}
