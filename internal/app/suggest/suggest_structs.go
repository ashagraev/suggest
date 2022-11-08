package suggest

type HighlightTextBlock struct {
  Text      string `json:"text"`
  Highlight bool   `json:"hl"`
}

type SuggestionItem struct {
  Weight     float32                `json:"weight"`
  Data       map[string]interface{} `json:"data"`
  TextBlocks []*HighlightTextBlock  `json:"text"`
}

type Pagination struct {
  PageNumber      int `json:"page_number"`
  TotalPagesCount int `json:"total_pages_count"`
  TotalItemsCount int `json:"total_items_count"`
}

type Response struct {
  Suggestions []*SuggestionItem `json:"suggestions"`
  Pagination  *Pagination       `json:"pagination"`
}

type Payload struct {
  OriginalPart   string
  NormalizedPart string
  Classes        map[string]bool
  ExcludeClasses map[string]bool
}
