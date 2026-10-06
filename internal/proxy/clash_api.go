package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// ProxyGroup represents a Clash / Mihomo selector or URLTest group.
type ProxyGroup struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"` // "Selector", "URLTest", "Fallback"
	Now     string       `json:"now"`
	All     []string     `json:"all"`
	History []DelayEntry `json:"history,omitempty"`
}

// DelayEntry records recent latency history for a proxy node.
type DelayEntry struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

// GetProxyGroups queries Clash REST API for active proxy groups and nodes.
func GetProxyGroups() ([]ProxyGroup, error) {
	resp, err := clashHTTPClient.Get(config.ClashAPI + "/proxies")
	if err != nil {
		return nil, fmt.Errorf("failed to reach Clash API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Clash API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Proxies map[string]struct {
			Name    string       `json:"name"`
			Type    string       `json:"type"`
			Now     string       `json:"now"`
			All     []string     `json:"all"`
			History []DelayEntry `json:"history"`
		} `json:"proxies"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse proxies JSON: %w", err)
	}

	var groups []ProxyGroup
	for name, p := range raw.Proxies {
		// Include selectors and groups with sub-nodes
		if len(p.All) > 0 || p.Type == "Selector" || p.Type == "URLTest" || p.Type == "Fallback" {
			groups = append(groups, ProxyGroup{
				Name:    name,
				Type:    p.Type,
				Now:     p.Now,
				All:     p.All,
				History: p.History,
			})
		}
	}

	// Sort groups alphabetically
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups, nil
}

// SelectProxyNode sets the active node in a selector group.
func SelectProxyNode(groupName, nodeName string) error {
	payload, err := json.Marshal(map[string]string{"name": nodeName})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/proxies/%s", config.ClashAPI, url.PathEscape(groupName))
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := clashHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to switch node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to switch node (HTTP %d): %s", resp.StatusCode, string(body))
	}

	logger.Get().Infof("proxy", "Switched group %s to node %s", groupName, nodeName)
	return nil
}

// TestNodeDelay tests latency for a specific proxy node.
func TestNodeDelay(nodeName string) (int, error) {
	testURL := "http://www.gstatic.com/generate_204"
	endpoint := fmt.Sprintf("%s/proxies/%s/delay?timeout=5000&url=%s", config.ClashAPI, url.PathEscape(nodeName), url.QueryEscape(testURL))

	resp, err := clashHTTPClient.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("ping test failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Delay   int    `json:"delay"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Message != "" && result.Delay <= 0 {
		return 0, fmt.Errorf("%s", result.Message)
	}

	return result.Delay, nil
}

// GetClashVersion queries active core version.
func GetClashVersion() (string, error) {
	resp, err := clashHTTPClient.Get(config.ClashAPI + "/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Version string `json:"version"`
		Premium bool   `json:"premium"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return strings.TrimSpace(res.Version), nil
}
