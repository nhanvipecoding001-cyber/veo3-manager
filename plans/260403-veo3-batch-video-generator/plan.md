# Veo3 Batch Video Generator — Implementation Plan

**Date:** 2026-04-03 | **Status:** Planning

## Project Summary

Windows desktop app automating batch video generation via Google Labs. User enters prompts, app controls Chrome to submit them sequentially, polls for completion, downloads videos. Built with Go/Wails v2 backend + React/TypeScript frontend.

## Tech Stack

Go 1.21+ | Wails v2 | go-rod/rod + stealth | modernc.org/sqlite | React 18 | TypeScript | Vite 5 | Tailwind v4 | Zustand 4 | Lucide React

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 1 | Project Scaffolding & Core Infrastructure | `[ ]` Planned | [phase-01-scaffolding.md](./phase-01-scaffolding.md) |
| 2 | Chrome Automation Engine | `[ ]` Planned | [phase-02-chrome-automation.md](./phase-02-chrome-automation.md) |
| 3 | Video Generation Pipeline | `[ ]` Planned | [phase-03-video-pipeline.md](./phase-03-video-pipeline.md) |
| 4 | Queue Management System | `[ ]` Planned | [phase-04-queue-management.md](./phase-04-queue-management.md) |
| 5 | Frontend — Layout & Navigation | `[ ]` Planned | [phase-05-frontend-layout.md](./phase-05-frontend-layout.md) |
| 6 | Frontend — Queue Page | `[ ]` Planned | [phase-06-frontend-queue.md](./phase-06-frontend-queue.md) |
| 7 | Frontend — Dashboard & History | `[ ]` Planned | [phase-07-frontend-dashboard-history.md](./phase-07-frontend-dashboard-history.md) |
| 8 | Frontend — Settings & Polish | `[ ]` Planned | [phase-08-frontend-settings-polish.md](./phase-08-frontend-settings-polish.md) |

## Research

- [Stack Research](./research/researcher-01-report.md)
- [Implementation Patterns](./research/researcher-02-report.md)

## Key Constraints

- No public API — must automate real Chrome (Fact #1)
- Anti-bot stealth required (Fact #3)
- Slate.js requires CDP `Input.insertText` (Fact #5)
- Video download requires redirect capture in Chrome tab (Fact #7)
