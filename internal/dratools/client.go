package dratools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DDBJClient struct {
	BaseURL     string
	HTTPClient  *http.Client
	ReadTimeout time.Duration
}

func NewDDBJClient(baseURL string) *DDBJClient {
	if baseURL == "" {
		baseURL = defaultDDBJSearchURL
	}
	return &DDBJClient{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		ReadTimeout: 30 * time.Second,
	}
}

func (c *DDBJClient) FetchResourceRecord(resourceType, accession string) (Record, error) {
	var record Record
	endpoint := fmt.Sprintf("%s/entries/%s/%s.json", c.BaseURL, resourceType, url.PathEscape(accession))
	if err := c.fetchJSON("GET", endpoint, nil, &record); err != nil {
		return nil, err
	}
	return record, nil
}

func (c *DDBJClient) FetchDBLinks(resourceType, accession, target string) ([]map[string]any, error) {
	endpoint := fmt.Sprintf("%s/dblink/%s/%s", c.BaseURL, resourceType, url.PathEscape(accession))
	if target != "" {
		endpoint += "?target=" + url.QueryEscape(target)
	}
	var payload map[string]any
	if err := c.fetchJSON("GET", endpoint, nil, &payload); err != nil {
		return nil, err
	}
	return mapSlice(payload[DBXrefsKey]), nil
}

func (c *DDBJClient) FetchResourceRecordsBulk(resourceType string, accessions []string, includeDBXrefs bool) (map[string]Record, error) {
	records := map[string]Record{}
	for start := 0; start < len(accessions); start += 1000 {
		end := start + 1000
		if end > len(accessions) {
			end = len(accessions)
		}
		chunk, err := c.fetchResourceRecordsBulkChunk(resourceType, accessions[start:end], includeDBXrefs)
		if err != nil {
			return nil, err
		}
		for key, record := range chunk {
			records[key] = record
		}
	}
	return records, nil
}

func (c *DDBJClient) fetchResourceRecordsBulkChunk(resourceType string, accessions []string, includeDBXrefs bool) (map[string]Record, error) {
	endpoint := fmt.Sprintf("%s/entries/%s/bulk?includeDbXrefs=%t", c.BaseURL, resourceType, includeDBXrefs)
	body := map[string]any{"ids": accessions}
	var payload map[string]any
	if err := c.fetchJSON("POST", endpoint, body, &payload); err != nil {
		return nil, err
	}
	out := map[string]Record{}
	for _, entry := range mapSlice(payload["entries"]) {
		record := Record(entry)
		if accession := recordAccession(record); accession != "" {
			out[accession] = record
		}
	}
	return out, nil
}

func (c *DDBJClient) FetchDBLinkCounts(items []map[string]string) (map[string]map[string]int, error) {
	out := map[string]map[string]int{}
	for start := 0; start < len(items); start += 100 {
		end := start + 100
		if end > len(items) {
			end = len(items)
		}
		chunk, err := c.fetchDBLinkCountsChunk(items[start:end])
		if err != nil {
			return nil, err
		}
		for key, counts := range chunk {
			out[key] = counts
		}
	}
	return out, nil
}

func (c *DDBJClient) fetchDBLinkCountsChunk(items []map[string]string) (map[string]map[string]int, error) {
	endpoint := c.BaseURL + "/dblink/counts"
	body := map[string]any{"items": items}
	var payload map[string]any
	if err := c.fetchJSON("POST", endpoint, body, &payload); err != nil {
		return nil, err
	}
	out := map[string]map[string]int{}
	for _, item := range mapSlice(payload["items"]) {
		key := stringValue(item, TypeKey) + "\x00" + xrefAccession(item)
		counts := map[string]int{}
		if raw, ok := item["counts"].(map[string]any); ok {
			for typ, value := range raw {
				switch v := value.(type) {
				case float64:
					counts[typ] = int(v)
				case int:
					counts[typ] = v
				}
			}
		}
		out[key] = counts
	}
	return out, nil
}

func (c *DDBJClient) fetchJSON(method, endpoint string, body any, dest any) error {
	var r io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return wrapError("network", "failed to encode request JSON", err)
		}
		r = bytes.NewReader(payload)
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.ReadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, endpoint, r)
	if err != nil {
		return wrapError("network", "failed to build request "+endpoint, err)
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return wrapError("network", "failed to fetch "+endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return newError("not_found", "not found: "+endpoint)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newError("network", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, endpoint))
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(dest); err != nil {
		return wrapError("network", "invalid JSON from "+endpoint, err)
	}
	return nil
}
