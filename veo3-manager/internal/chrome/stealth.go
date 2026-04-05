package chrome

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/stealth"
)

const additionalStealth = `() => {
	Object.defineProperty(navigator, 'platform', { get: () => 'Win32' });
	Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => 8 });
	Object.defineProperty(navigator, 'deviceMemory', { get: () => 8 });
	Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });

	// Override permissions query
	const origQuery = window.navigator.permissions.query;
	window.navigator.permissions.query = (parameters) => {
		if (parameters.name === 'notifications') {
			return Promise.resolve({ state: Notification.permission });
		}
		return origQuery(parameters);
	};
}`

func (bm *BrowserManager) NewStealthPage() (*rod.Page, error) {
	if bm.browser == nil {
		return nil, ErrNotConnected
	}

	page, err := stealth.Page(bm.browser)
	if err != nil {
		return nil, err
	}

	// Apply additional stealth overrides
	page.MustEvalOnNewDocument(additionalStealth)

	return page, nil
}

var ErrNotConnected = &BrowserError{Message: "browser not connected"}

type BrowserError struct {
	Message string
}

func (e *BrowserError) Error() string {
	return e.Message
}
