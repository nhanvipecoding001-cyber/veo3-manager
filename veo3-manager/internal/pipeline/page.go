package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ensurePage gets a page with a fresh project editor ready
func (p *Pipeline) ensurePage() (*rod.Page, error) {
	fmt.Println("[Pipeline] ensurePage: finding or creating page...")

	browser := p.browserMgr.Browser()
	if browser == nil {
		return nil, fmt.Errorf("browser not connected")
	}

	labsPage, err := p.findLabsPage(browser)
	if err != nil {
		return nil, err
	}

	// Always navigate to Flow homepage for a fresh start
	fmt.Println("[Pipeline] Navigating to Flow homepage...")
	if err := labsPage.Navigate("https://labs.google/fx/vi/tools/flow"); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}
	time.Sleep(5 * time.Second)
	fmt.Println("[Pipeline] Page loaded, creating new project...")

	if err := p.ensureProjectEditor(labsPage); err != nil {
		return nil, fmt.Errorf("failed to open project editor: %w", err)
	}

	p.activePage = labsPage
	return labsPage, nil
}

// findLabsPage finds an existing labs.google page or creates a new stealth page
func (p *Pipeline) findLabsPage(browser *rod.Browser) (*rod.Page, error) {
	pages, err := browser.Pages()
	if err == nil {
		for _, pg := range pages {
			info, err := pg.Info()
			if err == nil && strings.Contains(info.URL, "labs.google") {
				return pg, nil
			}
		}
	}

	fmt.Println("[Pipeline] No labs page found, creating new stealth page...")
	page, err := p.browserMgr.NewStealthPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create stealth page: %w", err)
	}
	return page, nil
}

// ensureProjectEditor checks if the page has the Slate editor, if not clicks "New project"
func (p *Pipeline) ensureProjectEditor(page *rod.Page) error {
	fmt.Println("[Pipeline] Checking if editor exists...")

	// Check if editor already exists (short timeout)
	if _, err := page.Timeout(3 * time.Second).Element("[data-slate-editor='true']"); err == nil {
		fmt.Println("[Pipeline] Editor already present")
		return nil
	}

	fmt.Println("[Pipeline] No editor found, looking for 'New project' button...")

	// Close any popup/banner first
	if closeBtn, err := page.Timeout(2 * time.Second).Element("button[aria-label='Close'], .sc-531cc04c-10"); err == nil {
		_ = closeBtn.Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(1 * time.Second)
	}

	// Click "Dự án mới" (New project) button using JavaScript for reliability
	clicked, err := page.Eval(`() => {
		const btns = Array.from(document.querySelectorAll('button'));
		const btn = btns.find(b => b.textContent.includes('Dự án mới') || b.textContent.includes('New project'));
		if (btn) {
			btn.scrollIntoView({block: 'center'});
			btn.click();
			return true;
		}
		return false;
	}`)
	if err != nil {
		return fmt.Errorf("failed to find buttons: %w", err)
	}
	if !clicked.Value.Bool() {
		return fmt.Errorf("'New project' button not found on page")
	}

	fmt.Println("[Pipeline] Clicked 'New project', waiting for editor...")
	time.Sleep(4 * time.Second)

	if _, err := page.Timeout(15 * time.Second).Element("[data-slate-editor='true']"); err != nil {
		return fmt.Errorf("editor not found after creating project: %w", err)
	}
	fmt.Println("[Pipeline] Editor ready!")
	return nil
}

func extractProjectIdFromURL(url string) string {
	parts := strings.Split(url, "/project/")
	if len(parts) < 2 {
		return ""
	}
	id := parts[1]
	if idx := strings.IndexAny(id, "/?#"); idx > 0 {
		id = id[:idx]
	}
	return id
}
