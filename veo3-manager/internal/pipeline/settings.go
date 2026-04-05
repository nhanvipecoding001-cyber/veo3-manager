package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ConfigureSettings sets video mode, aspect ratio, model, and output count on the Google Labs page
func ConfigureSettings(page *rod.Page, aspectRatio, model string, outputCount int) error {
	fmt.Printf("[Settings] Configuring: AR=%s, Model=%s, Count=%d\n", aspectRatio, model, outputCount)

	if err := openSettingsDropdown(page); err != nil {
		return fmt.Errorf("open settings: %w", err)
	}
	time.Sleep(1500 * time.Millisecond)

	if err := clickVideoTab(page); err != nil {
		fmt.Println("[Settings] Warning: click video tab:", err)
	}
	time.Sleep(1000 * time.Millisecond)

	// After clicking VIDEO tab, dropdown may close. Re-open if needed.
	if !isMenuOpen(page) {
		fmt.Println("[Settings] Menu closed after VIDEO tab, re-opening...")
		if err := openSettingsDropdown(page); err != nil {
			fmt.Println("[Settings] Warning: re-open dropdown:", err)
		}
		time.Sleep(1500 * time.Millisecond)
	}

	if err := selectAspectRatio(page, aspectRatio); err != nil {
		fmt.Println("[Settings] Warning: aspect ratio:", err)
	} else {
		fmt.Printf("[Settings] Aspect ratio set to %s\n", aspectRatio)
	}
	time.Sleep(500 * time.Millisecond)

	if err := selectOutputCount(page, outputCount); err != nil {
		fmt.Println("[Settings] Warning: output count:", err)
	} else {
		fmt.Printf("[Settings] Output count set to x%d\n", outputCount)
	}
	time.Sleep(500 * time.Millisecond)

	if model != "" {
		if !isMenuOpen(page) {
			fmt.Println("[Settings] Menu closed before model selection, re-opening...")
			if err := openSettingsDropdown(page); err != nil {
				fmt.Println("[Settings] Warning: re-open for model:", err)
			}
			time.Sleep(1500 * time.Millisecond)
		}
		if err := selectVideoModel(page, model); err != nil {
			fmt.Println("[Settings] Warning: model selection:", err)
		} else {
			fmt.Printf("[Settings] Model set to %s\n", model)
		}
		time.Sleep(500 * time.Millisecond)
	}

	closeDropdown(page)
	fmt.Println("[Settings] Configuration complete")
	return nil
}

func isMenuOpen(page *rod.Page) bool {
	_, err := page.Timeout(1 * time.Second).Element("[role='menu']")
	return err == nil
}

func closeDropdown(page *rod.Page) {
	_ = proto.InputDispatchKeyEvent{
		Type:                  proto.InputDispatchKeyEventTypeKeyDown,
		Key:                   "Escape",
		Code:                  "Escape",
		WindowsVirtualKeyCode: 27,
	}.Call(page)
	time.Sleep(300 * time.Millisecond)
	cdpClick(page, 400, 300)
	time.Sleep(500 * time.Millisecond)
}

func openSettingsDropdown(page *rod.Page) error {
	buttons, err := page.Timeout(5 * time.Second).Elements("button[aria-haspopup='menu']")
	if err != nil {
		return err
	}

	for _, btn := range buttons {
		text, err := btn.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(text, "crop_") && !strings.Contains(text, "Banana") &&
			!strings.Contains(text, "Veo") && !strings.Contains(text, "Video") {
			continue
		}
		_, _ = btn.Eval(`() => this.scrollIntoView({block: 'center'})`)
		time.Sleep(300 * time.Millisecond)

		shape, err := btn.Shape()
		if err != nil {
			return btn.Click(proto.InputMouseButtonLeft, 1)
		}
		box := shape.Box()
		x := box.X + box.Width/2
		y := box.Y + box.Height/2
		fmt.Printf("[Settings] Opening dropdown at (%.0f, %.0f)\n", x, y)
		cdpClick(page, x, y)
		return nil
	}
	return fmt.Errorf("settings dropdown button not found")
}

func clickVideoTab(page *rod.Page) error {
	tabs, err := page.Timeout(3 * time.Second).Elements("[role='tab']")
	if err != nil {
		return err
	}

	for _, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			continue
		}
		textLower := strings.ToLower(text)
		if !strings.Contains(textLower, "video") && !strings.Contains(text, "videocam") {
			continue
		}
		state, _ := tab.Attribute("data-state")
		if state != nil && *state == "active" {
			fmt.Println("[Settings] VIDEO tab already active")
			return nil
		}
		shape, err := tab.Shape()
		if err != nil {
			return tab.Click(proto.InputMouseButtonLeft, 1)
		}
		box := shape.Box()
		cdpClick(page, box.X+box.Width/2, box.Y+box.Height/2)
		return nil
	}
	return fmt.Errorf("VIDEO tab not found")
}

// cdpClick performs a reliable CDP mouse click at coordinates
func cdpClick(page *rod.Page, x, y float64) {
	_ = proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMousePressed,
		X: x, Y: y, Button: proto.InputMouseButtonLeft, ClickCount: 1,
	}.Call(page)
	_ = proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMouseReleased,
		X: x, Y: y, Button: proto.InputMouseButtonLeft, ClickCount: 1,
	}.Call(page)
}
