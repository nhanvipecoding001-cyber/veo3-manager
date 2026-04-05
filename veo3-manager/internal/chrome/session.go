package chrome

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

const labsURL = "https://labs.google/fx/vi/tools/flow"

// AuthInfo holds token and projectId extracted from the page
type AuthInfo struct {
	Token     string
	ProjectId string
}

func (bm *BrowserManager) ExtractAuth(page *rod.Page) (*AuthInfo, error) {
	info, err := page.Info()
	if err != nil || !strings.Contains(info.URL, "labs.google") {
		if err := page.Navigate(labsURL); err != nil {
			return nil, fmt.Errorf("failed to navigate to labs: %w", err)
		}
		page.MustWaitStable()
		time.Sleep(2 * time.Second)
	}

	result, err := page.Eval(`() => {
		const el = document.getElementById('__NEXT_DATA__');
		if (el) return el.textContent;
		if (window.__NEXT_DATA__) return JSON.stringify(window.__NEXT_DATA__);
		return null;
	}`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval __NEXT_DATA__: %w", err)
	}

	raw := result.Value.Str()
	if raw == "" {
		return nil, fmt.Errorf("__NEXT_DATA__ is empty — user may not be logged in")
	}

	authInfo, err := parseAuthFromNextData(raw)
	if err != nil {
		return nil, err
	}

	bm.mu.Lock()
	bm.token = authInfo.Token
	bm.tokenTime = time.Now()
	bm.projectId = authInfo.ProjectId
	bm.mu.Unlock()

	return authInfo, nil
}

func (bm *BrowserManager) RefreshToken(page *rod.Page) (string, error) {
	if err := page.Reload(); err != nil {
		return "", err
	}
	page.MustWaitStable()
	time.Sleep(2 * time.Second)

	auth, err := bm.ExtractAuth(page)
	if err != nil {
		return "", err
	}
	return auth.Token, nil
}

func (bm *BrowserManager) IsTokenValid() bool {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.token != "" && time.Since(bm.tokenTime) < 45*time.Minute
}

func (bm *BrowserManager) GetProjectId() string {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.projectId
}

func parseAuthFromNextData(raw string) (*AuthInfo, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__: %w", err)
	}

	token := findNestedValue(data, []string{"accessToken", "access_token", "token", "bearerToken"}, 0)
	if token == "" {
		return nil, fmt.Errorf("could not find auth token in __NEXT_DATA__")
	}

	projectId := findNestedValue(data, []string{"projectId"}, 0)

	return &AuthInfo{Token: token, ProjectId: projectId}, nil
}

// findNestedValue recursively searches for any of the given keys in nested JSON
func findNestedValue(data interface{}, keys []string, depth int) string {
	if depth > 10 {
		return ""
	}

	switch v := data.(type) {
	case map[string]interface{}:
		for _, key := range keys {
			if val, ok := v[key]; ok {
				if s, ok := val.(string); ok && len(s) > 0 {
					// For token keys, require minimum length
					if key == "projectId" || len(s) > 20 {
						return s
					}
				}
			}
		}
		for _, val := range v {
			if result := findNestedValue(val, keys, depth+1); result != "" {
				return result
			}
		}
	case []interface{}:
		for _, item := range v {
			if result := findNestedValue(item, keys, depth+1); result != "" {
				return result
			}
		}
	}
	return ""
}
