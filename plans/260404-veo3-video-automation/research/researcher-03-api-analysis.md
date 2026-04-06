# Google Labs Flow Video API Analysis

## Overview
Captured and verified complete video generation API flow via CDP automation on `labs.google/fx/vi/tools/flow`. All endpoints tested and working as of 2026-04-04.

## API Base
`https://aisandbox-pa.googleapis.com/v1`

## Authentication
- **Header**: `Authorization: Bearer ya29.a0Aa7MY...` (OAuth2 access token)
- **Token source**: `document.getElementById('__NEXT_DATA__')` → parse JSON → `props.pageProps.session.access_token`
- **Content-Type**: `text/plain;charset=UTF-8` (not application/json!)
- **Referer**: `https://labs.google/`
- **User-Agent**: Spoofed as Chrome 114 on macOS (stealth)

## Endpoints

### 1. Submit Video Generation
**`POST /v1/video:batchAsyncGenerateVideoText`**

Request body:
```json
{
  "mediaGenerationContext": { "batchId": "<uuid>" },
  "clientContext": {
    "projectId": "<uuid>",
    "tool": "PINHOLE",
    "userPaygateTier": "PAYGATE_TIER_NOT_PAID",
    "sessionId": ";<timestamp>",
    "recaptchaContext": {
      "token": "<recaptcha_token>",
      "applicationType": "RECAPTCHA_APPLICATION_TYPE_WEB"
    }
  },
  "requests": [{
    "aspectRatio": "VIDEO_ASPECT_RATIO_LANDSCAPE",
    "seed": 4938,
    "textInput": {
      "structuredPrompt": {
        "parts": [{ "text": "prompt text here" }]
      }
    },
    "videoModelKey": "veo_3_1_t2v_fast",
    "metadata": {}
  }],
  "useV2ModelConfig": true
}
```

Response:
```json
{
  "operations": [{
    "operation": { "name": "<media_id_uuid>" },
    "sceneId": "",
    "status": "MEDIA_GENERATION_STATUS_PENDING"
  }],
  "remainingCredits": 110,
  "workflows": [{ "name": "<workflow_id>", "metadata": {...}, "projectId": "<project_id>" }],
  "media": [{ "name": "<media_id>", ... }]
}
```

**Key observations**:
- Each request has unique `seed` (random int)
- `batchId` groups multiple videos from same prompt
- With `outputsPerPrompt: 2`, TWO separate POST calls are made (one per video)
- Each call gets its own `media_id` (operation name)
- Model: `veo_3_1_t2v_fast` (only working model currently)
- Aspect ratios: `VIDEO_ASPECT_RATIO_LANDSCAPE` (16:9), `VIDEO_ASPECT_RATIO_PORTRAIT` (9:16)
- reCAPTCHA token included but optional (API works without it)
- Credits: 20 per video (was 130, after 2 videos = 90)

### 2. Poll Status
**`POST /v1/video:batchCheckAsyncVideoGenerationStatus`**

Request body:
```json
{
  "media": [
    { "name": "<media_id>", "projectId": "<project_id>" },
    { "name": "<media_id_2>", "projectId": "<project_id>" }
  ]
}
```

Response (pending):
```json
{
  "media": [{
    "name": "<media_id>",
    "mediaMetadata": {
      "mediaStatus": { "mediaGenerationStatus": "MEDIA_GENERATION_STATUS_PENDING" }
    },
    "video": {
      "generatedVideo": {
        "seed": 4938, "model": "veo_3_1_t2v_fast",
        "aspectRatio": "VIDEO_ASPECT_RATIO_LANDSCAPE"
      }
    }
  }]
}
```

Response (success): same structure but `mediaGenerationStatus: "MEDIA_GENERATION_STATUS_SUCCESSFUL"` and `video.operation.name` field appears.

**Polling pattern**: every ~10s. Video generation takes 40-60s typically.

### 3. Download Video
**NOT via API.** Download requires browser cookies.

**Redirect URL**: `https://labs.google/fx/api/trpc/media.getMediaUrlRedirect?name=<media_id>`

This redirects to signed GCS URL:
`https://storage.googleapis.com/ai-sandbox-videofx/video/<media_id>?GoogleAccessId=labs-ai-sandbox-videoserver-prod@system.gserviceaccount.com&Expires=<ts>&Signature=<sig>`

**Download method**: 
1. Use browser `fetch()` or `XMLHttpRequest` from page context (needs cookies)
2. Get final GCS URL from `response.url` after redirect
3. Download from GCS URL directly via HTTP (signed URL, no auth needed)
4. File is `video/mp4`, ~7-8MB

### 4. Other Endpoints
- `GET /v1/credits?key=<api_key>` — check remaining credits
- `POST /v1:checkAppAvailability` — check if Flow is available
- `POST /v1:fetchUserRecommendations` — UI recommendations
- `PATCH /v1/flowWorkflows/<id>` — update workflow metadata (auto, not needed)
- `POST /v1:batchLog` / `POST /v1/flow:batchLogFrontendEvents` — analytics (not needed)

## UI Automation Details

### Model Selection (Dropdown)
- Button: `button[aria-haspopup="menu"]` containing `crop_` text
- Structure: tabs with `role="tab"` and `data-state="active/inactive"`
  - Row 1: `imageHình ảnh` | `videocamVideo` (IMAGE/VIDEO toggle)
  - Row 2: `crop_freeKhung hình` | `chrome_extensionThành phần` (Frame/Component)
  - Row 3: `crop_9_169:16` | `crop_16_916:9` (aspect ratio)
  - Row 4: `x1` | `x2` | `x3` | `x4` (output count)
  - Row 5: Model sub-dropdown button `Veo 3.1 - Fastarrow_drop_down`
- Click VIDEO tab → menu closes → model button updates to show "Video" mode

### Prompt Input
- Slate.js editor: `[data-slate-editor]`
- Must use CDP `Input.insertText` (keyboard events don't work)
- Clear existing: select all + Backspace

### Create Button
- Button at y > 700: contains `arrow_forwardTạo` or `add_2Tạo`
- Must use CDP mouse click (dispatchMouseEvent) for reliability

## Unresolved Questions
- reCAPTCHA: included in requests but seems optional. May become required under heavy use.
- Model sub-dropdown: `Veo 3.1 - Fast` shown but clicking VIDEO tab auto-selects it.
- `useV2ModelConfig: true` — unclear what v1 vs v2 differences are.
