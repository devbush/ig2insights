# Browse Account Reels - Design

## Overview

Implement the "Browse an account's reels" feature that allows users to browse, select, and process multiple reels from an Instagram account.

## User Flow

```
1. User selects "Browse an account's reels"
2. Prompt: "Enter username:"
3. Prompt: "Sort by:" → Top / Latest
4. Display first 10 reels with multi-select checkboxes
5. User selects reels, can load more pages, change sort
6. User selects "Continue with X selected"
7. Prompt: "What to download?" (multi-select)
8. Process all selected reels with chosen options
```

## Reel Display Format

```
[ ] "Caption truncated to ~30 chars..."  Jan 15  👁 12.3K  ❤️ 1.2K  💬 45  ↗️ 230
```

**Number formatting:**
- Under 1K: exact (e.g., `892`)
- 1K-999K: with K (e.g., `12.3K`)
- 1M+: with M (e.g., `1.2M`)

## Pagination

- Fetch 10 reels per page
- "Load more" appends next 10 (selections preserved)
- "Change sort" clears list, resets to page 1, clears selections
- Show current sort in menu

**Menu structure:**
```
[ ] Reel 1...
[ ] Reel 2...
...
[ ] Reel 10...
────────────────
[Load more]
[Change sort (Latest)]
[Continue with 3 selected]
```

**Edge cases:**
- No more reels: hide "Load more"
- No reels found: show "No reels found for @username"
- API error: show error, offer retry

## Output Options

```
What to download for 3 selected reels?

[x] Transcript (text)
[ ] Audio (WAV)
[ ] Video (MP4)
[ ] Thumbnail (JPG)

[Start processing]
```

- Transcript checked by default
- At least one option must be selected

## Processing

- Process reels sequentially (avoid rate limiting)
- Show progress: `Processing 1/3: DQe6auWjUI-...`
- Use existing `TranscribeService.Transcribe()` with appropriate options
- Output files go to current directory (or `--dir` if specified)
- Show summary: `Completed 3 reels. Files saved to ./`

**Error handling:**
- If one reel fails, continue with others
- Show summary: `Completed 2/3 reels. 1 failed: [error]`

## Technical Components

**Existing infrastructure:**
- `BrowseService` - has `ListReels(ctx, username, sort, limit)` and `GetAccount(ctx, username)`
- `tui` package for interactive menus
- Domain models: `Reel`, `Account` with stats fields

**New/modified components:**
1. `runAccountInteractive()` - Replace stub with full implementation
2. `ReelSelector` TUI component - Multi-select list with pagination
3. `OutputOptionsSelector` TUI component - Multi-select for output types
4. Format helpers - Number formatting, reel line display

## Files to Modify

| File | Changes |
|------|---------|
| `internal/adapters/cli/root.go` | Replace `runAccountInteractive()` stub |
| `internal/adapters/cli/tui/reel_selector.go` | **New** - Multi-select reel list |
| `internal/adapters/cli/tui/output_selector.go` | **New** - Output options selector |
| `internal/adapters/cli/tui/format.go` | **New** - Formatting helpers |
| `internal/adapters/cli/app.go` | Ensure `BrowseService` available |

## Testing

- Unit tests for number formatting
- Unit tests for reel line formatting
- Manual testing for TUI interactions
