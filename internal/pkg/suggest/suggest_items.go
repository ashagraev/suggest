package suggest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"log"
	"main/pkg/utils"
	"os"
	"strconv"
	"strings"
)

type Item struct {
	Weight         float32
	OriginalText   string
	NormalizedText string
	Data           map[string]interface{}
}

func NewItem(line string, policy *bluemonday.Policy) (*Item, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%d tab-separated fields, 3 expected", len(parts))
	}
	weight, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("cannot interpret %q as float", parts[1])
	}
	data := map[string]interface{}{}
	if err := json.Unmarshal([]byte(parts[2]), &data); err != nil {
		return nil, fmt.Errorf("cannot parse data json: %v", err)
	}
	return &Item{
		Weight:         float32(weight),
		NormalizedText: utils.NormalizeString(parts[0], policy),
		OriginalText:   parts[0],
		Data:           data,
	}, nil
}

func LoadItems(inputFilePath string, policy *bluemonday.Policy) ([]*Item, error) {
	file, err := os.Open(inputFilePath)
	if err != nil {
		return nil, err
	}
	var items []*Item
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		item, err := NewItem(line, policy)
		if err != nil {
			return nil, fmt.Errorf("error processing line #%d: %v", lineNumber, err)
		}
		items = append(items, item)
		lineNumber++
		if lineNumber%100000 == 0 {
			log.Printf("read %d lines", lineNumber)
		}
	}
	return items, nil
}
