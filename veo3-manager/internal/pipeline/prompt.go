package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// ClearEditor selects all text in the Slate.js editor and deletes it
func ClearEditor(page *rod.Page) error {
	editor, err := page.Element("[data-slate-editor='true']")
	if err != nil {
		return fmt.Errorf("slate editor not found: %w", err)
	}

	if err := editor.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	page.KeyActions().Press(input.ControlLeft).Type(input.KeyA).MustDo()
	time.Sleep(100 * time.Millisecond)
	page.KeyActions().Press(input.Backspace).MustDo()
	time.Sleep(200 * time.Millisecond)

	return nil
}

// InsertPrompt enters text into the Slate.js editor using CDP Input.insertText
func InsertPrompt(page *rod.Page, text string) error {
	editor, err := page.Element("[data-slate-editor='true']")
	if err != nil {
		return fmt.Errorf("slate editor not found: %w", err)
	}

	if err := editor.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	if err := (proto.InputInsertText{Text: text}).Call(page); err != nil {
		return fmt.Errorf("failed to insert text via CDP: %w", err)
	}
	return nil
}

// ClickCreate finds and clicks the Create button (y > 680px to avoid other buttons)
func ClickCreate(page *rod.Page) error {
	buttons, err := page.Elements("button")
	if err != nil {
		return fmt.Errorf("failed to find buttons: %w", err)
	}

	for _, btn := range buttons {
		text, err := btn.Text()
		if err != nil {
			continue
		}

		textLower := strings.ToLower(text)
		if !strings.Contains(text, "Tạo") &&
			!strings.Contains(textLower, "create") &&
			!strings.Contains(text, "arrow_forward") {
			continue
		}

		shape, err := btn.Shape()
		if err != nil {
			continue
		}
		box := shape.Box()
		if box.Y > 680 {
			cdpClick(page, box.X+box.Width/2, box.Y+box.Height/2)
			return nil
		}
	}

	return fmt.Errorf("Create button not found (y > 680)")
}
