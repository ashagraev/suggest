package suggest_merger

import (
  "context"
  "encoding/json"
  "fmt"
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
  Config        *Config
  SuggestClient *SuggestClient
}

type SuggestClient struct {
  httpClient *http.Client
}

func NewSuggestClient() *SuggestClient {
  return &SuggestClient{
    httpClient: &http.Client{
      Timeout: time.Second * 10,
    },
  }
}

func get(requestURL string, headers http.Header, client *http.Client) (int, []byte, http.Header, error) {
  req, err := http.NewRequest("GET", requestURL, nil)
  if err != nil {
    return 0, nil, nil, fmt.Errorf("cannot create request for the url %s: %v", requestURL, err)
  }
  req.Header = headers
  res, err := client.Do(req)
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

    results := make([]*suggest.PaginatedSuggestResponse, len(h.Config.SuggestShardsUrls))
    versions := make([]uint64, len(h.Config.SuggestShardsUrls))

    for i, suggestShardUrl := range h.Config.SuggestShardsUrls {
      i, suggestShardUrl := i, suggestShardUrl // https://golang.org/doc/faq#closures_and_goroutines

      newSuggestShardUrl, err := url.Parse(suggestShardUrl)
      if err != nil {
        log.Fatal(err)
      }

      query.Add("api-version", "2")
      newSuggestShardUrl.RawQuery = query.Encode()

      g.Go(func() error {
        _, result, header, err := get(newSuggestShardUrl.String(), r.Header, h.SuggestClient.httpClient)

        if err != nil {
          return err
        }

        versions[i] = getSuggestVersion(header)

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
    if version > maxVersion {
      paginatedResp = results[i]
      maxVersion = version
    }
  }

  pagingParameters := suggest.NewPagingParameters(r.URL.Query())
  if pagingParameters.PaginationOn {
    network.ReportSuccessData(w, paginatedResp)
  } else {
    network.ReportSuccessData(w, &suggest.SuggestResponse{Suggestions: paginatedResp.Suggestions})
  }
}

func (h *Handler) HandleMergerHealthRequest(w http.ResponseWriter, _ *http.Request) {
  network.ReportSuccessMessage(w, "OK")
}
