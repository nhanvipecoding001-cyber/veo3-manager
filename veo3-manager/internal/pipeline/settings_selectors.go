package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// selectAspectRatio selects aspect ratio tab by matching text
func selectAspectRatio(page *rod.Page, ratio string) error {
	tabs, err := page.Timeout(3 * time.Second).Elements("[role='tab']")
	if err != nil {
		return fmt.Errorf("no tabs found: %w", err)
	}

	for _, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(text, ratio) {
			continue
		}
		state, _ := tab.Attribute("data-state")
		if state != nil && *state == "active" {
			return nil
		}
		return tab.Click(proto.InputMouseButtonLeft, 1)
	}
	return fmt.Errorf("aspect ratio tab '%s' not found", ratio)
}

// selectOutputCount selects output count tab (x1, x2, x3, x4)
func selectOutputCount(page *rod.Page, count int) error {
	countStr := "x" + strconv.Itoa(count)
	tabs, err := page.Timeout(3 * time.Second).Elements("[role='tab']")
	if err != nil {
		return err
	}

	for _, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) != countStr {
			continue
		}
		state, _ := tab.Attribute("data-state")
		if state != nil && *state == "active" {
			return nil
		}
		return tab.Click(proto.InputMouseButtonLeft, 1)
	}
	return fmt.Errorf("output count tab '%s' not found", countStr)
}

// selectVideoModel opens model sub-dropdown and selects target model
func selectVideoModel(page *rod.Page, model string) error {
	displayName := mapModelDisplayName(model)
	fmt.Printf("[Settings] Selecting model: %s -> display: %s\n", model, displayName)

	menu, err := page.Timeout(3 * time.Second).Element("[role='menu']")
	if err != nil {
		return fmt.Errorf("menu not found: %w", err)
	}

	// Click the model sub-dropdown button
	if err := clickModelDropdownButton(menu); err != nil {
		return err
	}
	time.Sleep(1500 * time.Millisecond)

	// Select target model from menuitems
	items, err := page.Timeout(3 * time.Second).Elements("[role='menuitem']")
	if err != nil {
		return fmt.Errorf("no menuitems found: %w", err)
	}

	for _, item := range items {
		text, err := item.Text()
		if err != nil {
			continue
		}
		if strings.Contains(text, displayName) {
			trimmed := strings.TrimSpace(text)
			if len(trimmed) > 40 {
				trimmed = trimmed[:40]
			}
			fmt.Printf("[Settings] Clicking model: %s\n", trimmed)
			return item.Click(proto.InputMouseButtonLeft, 1)
		}
	}
	return fmt.Errorf("model '%s' (display: '%s') not found in menu", model, displayName)
}

func clickModelDropdownButton(menu *rod.Element) error {
	buttons, err := menu.Elements("button")
	if err != nil {
		return err
	}
	for _, btn := range buttons {
		text, err := btn.Text()
		if err != nil {
			continue
		}
		if strings.Contains(text, "arrow_drop_down") || strings.Contains(text, "Veo") {
			fmt.Println("[Settings] Clicking model dropdown button")
			return btn.Click(proto.InputMouseButtonLeft, 1)
		}
	}
	return fmt.Errorf("model sub-dropdown button not found")
}

// mapModelDisplayName maps model API keys to display names in Google Labs UI
func mapModelDisplayName(model string) string {
	switch model {
	case "veo_3_1_t2v_lite", "lite":
		return "Lite"
	case "veo_3_1_t2v_fast", "veo_3_1_t2v_fast_ultra", "fast", "":
		return "Fast"
	case "veo_3_1_t2v_quality", "quality":
		return "Quality"
	default:
		return "Lite"
	}
}
