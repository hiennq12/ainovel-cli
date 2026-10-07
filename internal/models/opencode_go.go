package models

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Giá của gói OpenCode Go khác giá niêm yết trên OpenRouter (vd. deepseek-v4-pro),
// nên lấy riêng từ models.dev — cơ sở dữ liệu model do chính OpenCode duy trì,
// provider key "opencode-go". Chỉ ghi cache để tra cứu/so sánh; chưa nạp vào
// ModelRegistry nên không ảnh hưởng cách app tính chi phí.
const (
	modelsDevURL            = "https://models.dev/api.json"
	openCodeGoProviderKey   = "opencode-go"
	openCodeGoCacheFileName = "opencode-go-pricing.json"
	// api.json chứa mọi provider (~5MB), cần rộng hơn fetchTimeout của OpenRouter.
	modelsDevFetchTimeout = 60 * time.Second
)

// OpenCodeGoPrice là giá một model trong gói OpenCode Go (USD / 1M token).
type OpenCodeGoPrice struct {
	ModelEntry
	// Tiers là các bậc giá theo độ dài context (vd. vượt 256k token thì đắt hơn).
	Tiers []PriceTier `json:"tiers,omitempty"`
}

// PriceTier áp dụng khi context của request vượt ContextOver token.
type PriceTier struct {
	ContextOver         int     `json:"context_over"`
	InputCostPer1M      float64 `json:"input_cost_per_1m"`
	OutputCostPer1M     float64 `json:"output_cost_per_1m"`
	CacheReadCostPer1M  float64 `json:"cache_read_cost_per_1m"`
	CacheWriteCostPer1M float64 `json:"cache_write_cost_per_1m"`
}

type openCodeGoCache struct {
	FetchedAt time.Time         `json:"fetched_at"`
	Source    string            `json:"source"`
	Models    []OpenCodeGoPrice `json:"models"`
}

type modelsDevProvider struct {
	Models map[string]modelsDevModel `json:"models"`
}

type modelsDevModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cost *struct {
		modelsDevCost
		Tiers []struct {
			modelsDevCost
			Tier struct {
				Type string `json:"type"`
				Size int    `json:"size"`
			} `json:"tier"`
		} `json:"tiers"`
	} `json:"cost"`
	Limit struct {
		Context int `json:"context"`
		Output  int `json:"output"`
	} `json:"limit"`
}

type modelsDevCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// StartOpenCodeGoPricingRefresh làm mới cache giá OpenCode Go trong goroutine nền,
// cùng TTL 24h với models-cache.json; cache còn hạn thì không tải lại.
func StartOpenCodeGoPricingRefresh(cacheDir string) {
	if cacheDir == "" {
		return
	}
	go func() {
		if openCodeGoCacheFresh(cacheDir) {
			return
		}
		prices, err := fetchOpenCodeGoPrices()
		if err != nil {
			slog.Warn("Làm mới giá OpenCode Go thất bại", "module", "models", "err", err)
			return
		}
		data, err := json.MarshalIndent(openCodeGoCache{
			FetchedAt: time.Now(), Source: modelsDevURL, Models: prices,
		}, "", "  ")
		if err != nil {
			return
		}
		writeCacheFile(cacheDir, openCodeGoCacheFileName, data)
		slog.Info("Giá OpenCode Go đã cập nhật", "module", "models", "count", len(prices))
	}()
}

func openCodeGoCacheFresh(cacheDir string) bool {
	data, err := os.ReadFile(filepath.Join(cacheDir, openCodeGoCacheFileName))
	if err != nil {
		return false
	}
	var c openCodeGoCache
	if err := json.Unmarshal(data, &c); err != nil {
		return false
	}
	return time.Since(c.FetchedAt) <= cacheTTL
}

func fetchOpenCodeGoPrices() ([]OpenCodeGoPrice, error) {
	client := &http.Client{Timeout: modelsDevFetchTimeout}
	resp, err := client.Get(modelsDevURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev API returned %d", resp.StatusCode)
	}
	var all map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, err
	}
	raw, ok := all[openCodeGoProviderKey]
	if !ok {
		return nil, fmt.Errorf("models.dev không có provider %q", openCodeGoProviderKey)
	}
	return parseOpenCodeGoProvider(raw)
}

func parseOpenCodeGoProvider(raw json.RawMessage) ([]OpenCodeGoPrice, error) {
	var p modelsDevProvider
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	out := make([]OpenCodeGoPrice, 0, len(p.Models))
	for key, m := range p.Models {
		id := m.ID
		if id == "" {
			id = key
		}
		price := OpenCodeGoPrice{ModelEntry: ModelEntry{
			Provider:      openCodeGoProviderKey,
			ID:            id,
			Name:          m.Name,
			ContextWindow: m.Limit.Context,
			MaxTokens:     m.Limit.Output,
		}}
		if m.Cost != nil {
			price.InputCostPer1M = m.Cost.Input
			price.OutputCostPer1M = m.Cost.Output
			price.CacheReadCostPer1M = m.Cost.CacheRead
			price.CacheWriteCostPer1M = m.Cost.CacheWrite
			for _, t := range m.Cost.Tiers {
				if t.Tier.Type != "context" || t.Tier.Size <= 0 {
					continue
				}
				price.Tiers = append(price.Tiers, PriceTier{
					ContextOver:         t.Tier.Size,
					InputCostPer1M:      t.Input,
					OutputCostPer1M:     t.Output,
					CacheReadCostPer1M:  t.CacheRead,
					CacheWriteCostPer1M: t.CacheWrite,
				})
			}
		}
		out = append(out, price)
	}
	// map không có thứ tự; sắp theo ID để file cache ổn định giữa các lần tải.
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID) })
	return out, nil
}
