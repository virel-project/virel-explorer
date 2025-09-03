package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"virel-explorer/html"

	"github.com/virel-project/virel-blockchain/v2/config"
)

type coinpaprikaResponse struct {
	Quotes map[string]struct {
		Price     float64 `json:"price"`
		Volume24h float64 `json:"volume_24h"`
		Change    float64 `json:"percent_change_24h"`
	} `json:"quotes"`
}

func GetMarketInfo(supply uint64) (*html.MarketInfo, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	client.Transport = &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}

	// Create GET request
	req, err := http.NewRequest("GET", "https://api.coinpaprika.com/v1/tickers/vrl-virel", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code not ok: %d body: %s", resp.StatusCode, body)
	}

	res := coinpaprikaResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}
	usdq := res.Quotes["USD"]

	minf := &html.MarketInfo{}

	minf.Change = fmt.Sprintf("%.2f", usdq.Change) + "%"
	if usdq.Change >= 0 {
		minf.Change = "+" + minf.Change
	}
	minf.Price = usdq.Price

	minf.Supply = (float64(supply) / config.COIN)
	minf.Marketcap = minf.Price * (float64(supply) / config.COIN)

	return minf, nil
}
