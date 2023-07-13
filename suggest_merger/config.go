package suggest_merger

import (
  "encoding/json"
  "io/ioutil"
  "os"
)

type Config struct {
  SuggestShardsUrls []string `json:"suggest_shards_urls"`
}

func ReadConfig(configPath string) (*Config, error) {
  jsonFile, err := os.Open(configPath)
  if err != nil {
    return nil, err
  }
  defer jsonFile.Close()

  byteValue, _ := ioutil.ReadAll(jsonFile)
  config := &Config{}
  if err = json.Unmarshal(byteValue, config); err != nil {
    return nil, err
  }
  return config, nil
}
