package suggest_merger

import (
  "context"
  "encoding/json"
  "fmt"
  "github.com/hashicorp/go-retryablehttp"
  "golang.org/x/sync/errgroup"
  "io/ioutil"
  "log"
  "main/network"
  "main/suggest"
  "net/http"
  "net/url"
  "strconv"
  "time"
)

type Handler struct {
  Config            *Config
  SuggestClient     *SuggestClient
  SuggestShardsUrls []url.URL
}

func NewHandler(config *Config) (*Handler, error) {
  h := &Handler{
    SuggestClient: NewSuggestClient(),
    Config:        config,
  }
  if err := h.initSuggestShardsUrls(); err != nil {
    return nil, err
  }
  return h, nil
}

func (h *Handler) initSuggestShardsUrls() error {
  for _, suggestShardUrl := range h.Config.SuggestShardsUrls {
    shardUrl, err := url.Parse(suggestShardUrl)
    if err != nil {
      return err
    }
    h.SuggestShardsUrls = append(h.SuggestShardsUrls, *shardUrl)
  }
  return nil
}

type SuggestClient struct {
  httpClient *retryablehttp.Client
}

func NewSuggestClient() *SuggestClient {
  return &SuggestClient{
    httpClient: &retryablehttp.Client{
      RetryMax:     10,
      RetryWaitMin: 10 * time.Millisecond,
      HTTPClient: &http.Client{
        Timeout: time.Second * 10,
      },
      CheckRetry: retryablehttp.DefaultRetryPolicy,
      Backoff: retryablehttp.DefaultBackoff,
    },
  }
}

func (sc *SuggestClient) Get(requestURL string, headers http.Header) (int, []byte, http.Header, error) {
  req, err := retryablehttp.NewRequest("GET", requestURL, nil)
  if err != nil {
    return 0, nil, nil, fmt.Errorf("cannot create request for the url %s: %v", requestURL, err)
  }
  req.Header = headers
  res, err := sc.httpClient.Do(req)
  if err != nil {
    return 0, nil, nil, fmt.Errorf("cannot execute request for the url %s: %v", requestURL, err)
  }
  content, err := ioutil.ReadAll(res.Body)
  return res.StatusCode, content, res.Header, err
}

func getSuggestVersion(header http.Header) uint64 {
  v, _ := strconv.ParseUint(header.Get("Suggest-Version"), 0, 64)
  return v
}

func (h *Handler) HandleMergerSuggestRequest(w http.ResponseWriter, r *http.Request) {
  doRequests := func(ctx context.Context, query url.Values) (
    []*suggest.PaginatedSuggestResponse,
    []uint64,
    error,
  ) {
    g, ctx := errgroup.WithContext(ctx)

    results := make([]*suggest.PaginatedSuggestResponse, len(h.SuggestShardsUrls))
    versions := make([]uint64, len(h.SuggestShardsUrls))

    query.Add("api-version", "2")

    for i, suggestShardUrl := range h.SuggestShardsUrls {
      i, suggestShardUrl := i, suggestShardUrl // https://golang.org/doc/faq#closures_and_goroutines

      g.Go(func() error {
        suggestShardUrl.RawQuery = query.Encode()

        _, result, headers, err := h.SuggestClient.Get(suggestShardUrl.String(), r.Header)
        if err != nil {
          return err
        }

        versions[i] = getSuggestVersion(headers)

        paginatedResponse := &suggest.PaginatedSuggestResponse{}
        err = json.Unmarshal(result, paginatedResponse)
        if err != nil {
          return err
        }
        results[i] = paginatedResponse

        return nil
      })
    }
    if err := g.Wait(); err != nil {
      return nil, nil, err
    }
    return results, versions, nil
  }

  srcQuery := r.URL.Query()
  results, versions, err := doRequests(context.Background(), srcQuery)
  if err != nil {
    log.Println(err)
  }

  var paginatedResp *suggest.PaginatedSuggestResponse
  var maxVersion uint64
  for i, version := range versions {
    if version > maxVersion && len(results[i].Suggestions) > 0 {
      paginatedResp = results[i]
      maxVersion = version
    }
  }

  pagingParameters := suggest.NewPagingParameters(srcQuery)
  if pagingParameters.PaginationOn {
    network.ReportSuccessData(w, paginatedResp)
  } else {
    network.ReportSuccessData(w, &suggest.SuggestResponse{Suggestions: paginatedResp.Suggestions})
  }
}

func (h *Handler) HandleMergerHealthRequest(w http.ResponseWriter, _ *http.Request) {
  network.ReportSuccessMessage(w, "OK")
}
