# Phase 4: Fix UI Automation

## Context
- [API Analysis - UI Automation Details](research/researcher-03-api-analysis.md)
- [Codebase Scout](scout/scout-01-codebase.md)

## Overview
Fix `prompt.go` and `settings.go` for correct selectors and locale. Button says "Tao" not "Create" (Vietnamese locale). Settings dropdown needs VIDEO tab click before aspect ratio selection. Add CDP mouse click for Create button reliability.

## Key Insights
- Page is Vietnamese locale (`/fx/vi/tools/flow`). All button text is Vietnamese.
- Create button: contains `"Tao"` (with diacritics: `"Tạo"`), preceded by material icon text like `"arrow_forward"` or `"add_2"`
- Settings dropdown structure: IMAGE/VIDEO tabs -> Frame/Component tabs -> aspect ratio tabs -> output count tabs -> model sub-dropdown
- Must click VIDEO tab first -- otherwise settings are for IMAGE mode
- Slate.js editor selector `[data-slate-editor]` is correct
- CDP `Input.insertText` for prompt is correct
- Create button needs CDP dispatchMouseEvent for reliability (not rod Click)

## Requirements
1. Fix ClickCreate: search for "Tao" not "Create"
2. Add VIDEO tab switch in ConfigureSettings
3. Use CDP mouse events for Create button click
4. Make button detection locale-independent (fallback: position-based)
5. Fix aspect ratio tab text matching for Vietnamese

## Architecture

### Button Detection Strategy
Primary: match button text containing "Tao" (covers both `"Tạo"` and ASCII fallback).
Fallback: find button at y > 700 with material icon `arrow_forward` or `add_2`.
Final fallback: any button at y > 700 that is not disabled.

### VIDEO Tab Detection
Look for `role="tab"` containing text "Video" or "video" or the Vietnamese equivalent. The tab text from capture is `"videocamVideo"` (material icon + label).

## Related Code Files
- `veo3-manager/internal/pipeline/prompt.go` (FIX)
- `veo3-manager/internal/pipeline/settings.go` (FIX)

## Implementation Steps

### Step 1: Fix ClickCreate in prompt.go
```go
func ClickCreate(page *rod.Page) error {
    buttons, _ := page.Elements("button")
    for _, btn := range buttons {
        text, _ := btn.Text()
        // Match Vietnamese "Tạo" or ASCII "Tao"
        if !strings.Contains(text, "Tạo") && !strings.Contains(text, "Tao") {
            continue
        }
        shape, _ := btn.Shape()
        if shape.Box().Y > 680 {
            // Use CDP mouse click for reliability
            box := shape.Box()
            x := box.X + box.Width/2
            y := box.Y + box.Height/2
            return cdpClick(page, x, y)
        }
    }
    return fmt.Errorf("Create button not found")
}
```

### Step 2: Add cdpClick helper
```go
func cdpClick(page *rod.Page, x, y float64) error {
    proto.InputDispatchMouseEvent{
        Type: proto.InputDispatchMouseEventTypeMousePressed,
        X: x, Y: y, Button: proto.InputMouseButtonLeft, ClickCount: 1,
    }.Call(page)
    return proto.InputDispatchMouseEvent{
        Type: proto.InputDispatchMouseEventTypeMouseReleased,
        X: x, Y: y, Button: proto.InputMouseButtonLeft, ClickCount: 1,
    }.Call(page)
}
```

### Step 3: Add selectVideoTab in settings.go
Insert before aspect ratio selection in ConfigureSettings:
```go
func selectVideoTab(page *rod.Page) error {
    tabs, _ := page.Elements("[role='tab']")
    for _, tab := range tabs {
        text, _ := tab.Text()
        if strings.Contains(text, "Video") || strings.Contains(text, "video") {
            state, _ := tab.Attribute("data-state")
            if state != nil && *state == "active" {
                return nil // already selected
            }
            return tab.Click(proto.InputMouseButtonLeft, 1)
        }
    }
    return fmt.Errorf("VIDEO tab not found")
}
```

### Step 4: Update ConfigureSettings order
```go
func ConfigureSettings(page *rod.Page, aspectRatio, model string, outputCount int) error {
    openSettingsDropdown(page)
    page.MustWaitStable()
    selectVideoTab(page)        // NEW: must come first
    page.MustWaitStable()       // dropdown may re-render after VIDEO tab
    // Re-open dropdown if it closed after VIDEO tab click
    openSettingsDropdown(page)
    page.MustWaitStable()
    selectAspectRatio(page, aspectRatio)
    selectOutputCount(page, outputCount)
    // Model auto-selected as Veo 3.1 Fast when VIDEO tab clicked
    return nil
}
```

### Step 5: Fix aspect ratio tab text
Current code matches `"16:9"` directly. Real tab text is `"crop_16_916:9"` (icon + text). The `strings.Contains(text, ratio)` should still work since `"16:9"` is a substring. Verify and keep.

### Step 6: Close dropdown after configuration
Click outside the dropdown or press Escape to close it before proceeding to prompt entry.

## Todo
- [ ] Fix ClickCreate to match "Tao"/"Tạo" instead of "Create"
- [ ] Add cdpClick helper for reliable mouse events
- [ ] Add selectVideoTab function
- [ ] Update ConfigureSettings to call selectVideoTab first
- [ ] Handle dropdown close/reopen after VIDEO tab switch
- [ ] Add close-dropdown step after all settings configured
- [ ] Test with Vietnamese locale page

## Success Criteria
- ClickCreate finds and clicks the Vietnamese "Tạo" button
- VIDEO tab is selected before aspect ratio configuration
- Settings dropdown opens/closes reliably
- CDP mouse events work where rod Click fails

## Risk Assessment
- **Dropdown close on tab click**: Clicking VIDEO tab may close the dropdown. Must re-open it for subsequent settings. Add retry logic.
- **Locale changes**: If user switches to English, "Tạo" becomes "Create". Add both as fallback matches.
- **Material icon text in button**: `btn.Text()` returns icon ligature names + label (e.g., `"arrow_forwardTạo"`). Substring match handles this.

## Security Considerations
- No security impact. UI automation only interacts with the user's own browser session.

## Next Steps
Phase 5 polishes queue management and frontend for production use.
