# Upload Dashboard Wireframe

**Purpose:** Foundation design for the new TUI based on current screen capture analysis

**Date:** 2025-12-14

---

## Layout Overview

The dashboard uses a full-terminal layout with the following vertical structure:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ HEADER (1 row)                                                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│ STATS ROW (multi-column cards)                                          │
│ [Discovery] [Processing] [Progress] [Server's Jobs]                     │
│                                                                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│ LOG PANEL (expanding, scrollable)                                       │
│                                                                          │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. Header Section
- **Content:** Application name and version
- **Layout:** Single row, left-aligned
- **Example:** `immich-go v dev | Upload | Server http://localhost:2283 |  User: demo@immich.app`
- **Style:** Plain text, no border
-

### 2. Stats Row 
When the screen is wide enough, the stats row contains four cards side-by-side. If the terminal width is too narrow, the cards stack vertically.
The card are seprated by a vertical pipe and a single space gap when side-by-side, or a 1-line row gap when stacked.

#### 2.1 Card A: Discovery
**Purpose:** Show file system scan results

**Content:**
```
Source discovery:
Images            336  495.6 MB
Videos              4   54.5 MB
                               
Duplicates (local)  0       0 B
Already on server   0       0 B
Filtered (rules)    1    5.8 MB
Banned              0       0 B
Missing sidecar     0       0 B
```

**Data Points:**
- Images: count + size
- Videos: count + size
- Duplicates (local): count + size
- Already on server: count + size
- Filtered (rules): count + size
- Banned: count + size
- Missing sidecar: count + size

**Layout:** 2-column alignment (label left, values right)

---

#### 2.2 Card B: Processing
**Purpose:** Show actions done with the source

**Content:**
```
Source processing:
sidecars associated  340
Added to albums        5
Stacked               18
Tagged                25
Metadata updated       0
```

**Data Points:**
- Sidecars associated: count
- Added to albums: count
- Stacked: count
- Tagged: count
- Metadata updated: count

**Layout:** 2-column alignment (label left, count right)

---

#### 2.3 Card C: Progress
**Purpose:** Show upload progress statistics

**Content:**
```
Pending      340  490.1 MB
Processed       0      0 B
Discarded       1   5.8 MB
Errors          0      0 B
Total        341  495.9 MB
```


**Data Points:**
- Pending: count + size
- Processed: count + size
- Discarded: count + size
- Errors: count + size
- Total: count + size (summary)

**Layout:** 2-column alignment (label left, values right)

**Note:** "Total" row should be visually distinct (bold or different color)

#### 2.4 Card D: Server's Jobs
**Purpose:** Show upload jobs, 

**Content:**
```
Server operations:
Upload jobs:    ▂▂▃▃▄▄▄▅▅▆▆▇  10
Files uploaded: ▂▂▃▃▄▄▄▅▅▆▆▇  12 /s
Bandwidth:      ▂▂▃▃▄▄▄▅▅▆▆▇ 162 MB/s
```


---

### 3. Log Panel
**Purpose:** Real-time streaming log of upload operations

**Content:**
```
2025-12-14 16:38:13 INF uploaded successfully file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183507.jpg
2025-12-14 16:38:13 INF metadata updated file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_190957.jpg
2025-12-14 16:38:13 INF added to album file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_190957.jpg album=Duplicated album
2025-12-14 16:38:13 INF added to album file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_190957.jpg album=Pique-nique du 11 Août 2018
2025-12-14 16:38:13 INF tagged file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_190957.jpg tag=takeout-20240123T180723Z
2025-12-14 16:38:13 INF uploaded successfully file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183902.jpg
2025-12-14 16:38:13 INF metadata updated file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183855.jpg
2025-12-14 16:38:13 INF added to album file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183855.jpg album=Duplicated album
2025-12-14 16:38:13 INF added to album file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183855.jpg album=Pique-nique du 11 Août 2018
2025-12-14 16:38:13 INF tagged file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183855.jpg tag=takeout-20240123T180723Z
2025-12-14 16:38:13 INF metadata updated file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183507.jpg
2025-12-14 16:38:13 INF added to album file=takeout-20240123T180723Z-001:Takeout/Google Photos/DupAlbum/IMG_20180811_183507.jpg album=Duplicated album
```

**Features:**
- Scrollable (shows most recent entries)
- Color-coded log levels (INF=cyan, WRN=yellow, ERR=red)
- Timestamp + level + message format
- Auto-scroll to bottom for new entries
- Full terminal width
- Expands to fill remaining vertical space

**Data Format:**
- Timestamp: `YYYY-MM-DD HH:MM:SS`
- Level: `INF`/`WRN`/`ERR`
- Message: structured with keywords colored (file=, json=, matcher=)

---

## Layout Behavior

### Responsive Sizing
- **Cards in Stats Row:**
  - Minimum width: 38 characters per card
  - Column gap: 4 characters between cards
  - If terminal width < (38*4 + 4*3) = 164 chars: stack cards vertically
  - Otherwise: place side-by-side in a single row

- **Log Panel:**
  - Always full width
  - Height: Expands to fill remaining space after header + stats row
  - Row gap: 1 line between stats row and log panel

### Dynamic Updates
All sections update in real-time:
- **Discovery:** Updates as file scanning progresses
- **Processing:** Updates as server processes assets
- **Progress:** Updates continuously during upload
- **Server's Jobs:** Polls server every N seconds
- **Log:** Streams new entries as they occur

---

## Visual Design Principles

### Borders
- Lipgloss borders can't be used to pase a title. Later we can explore custom border rendering if needed.
- Title in border top-left position
- Consistent padding: 1 space inside borders

### Spacing
- Row gap between stats and logs: 1 line
- Column gap between stat cards: 4 spaces
- No gap between header and stats row

### Colors (Conceptual)
- Card borders: subtle gray/white
- Log levels: INF=cyan, WRN=yellow, ERR=red
- Keywords in logs: distinct colors (file=, json=, matcher=)
- Total row in Progress: bold or highlighted

### Alignment
- Header: left-aligned
- Card titles: embedded in border, top, left-aligned
- Card content: 2-column tables with right-aligned values
- Logs: left-aligned, monospaced

---

## Implementation Notes

### Current State
- Basic skeleton exists in `upload_dashboard.go`
- Event pipeline wired for inventory, jobs, logs
- Simplified to header + logs only (cards removed during debugging)

### Next Steps (MVP)
1. **Restore 4-card stats row** with proper width calculations
2. **Implement responsive stacking** (side-by-side vs vertical)
3. **Fix card border rendering** (resolve width calculation issues)
4. **Add color coding** for log levels and keywords
5. **Implement auto-scroll** for log panel
6. **Polish spacing and alignment**

### Future Enhancements (Post-MVP)
- Pause/resume controls in footer
- Interactive card focus/expansion
- Configurable log filtering
- Export/save logs
- Progress bars within cards
- Server job details expansion

---

## Data Flow

```
Business Logic (upload process)
    ↓
Event Publishers
    ↓
Event Stream (messages.Stream)
    ↓
Dashboard Model (Bubble Tea Update())
    ↓
Dashboard View (Bubble Tea View())
    ↓
Terminal Rendering
```

**Key Events:**
- `InventoryEvent`: Updates Discovery card
- `JobEvent`: Updates Server's Jobs card
- `ProgressEvent`: Updates Progress card (implied)
- `LogEvent`: Appends to Log panel
- `ProcessingEvent`: Updates Processing card (implied)

---

## Technical Constraints

### Layout Library
- **Lipgloss** for styling and layout
- `JoinHorizontal()` for card rows
- `PlaceVertical()` for full layout stacking
- `Place()` for width/height enforcement

### Width Calculations
- Terminal width via Bubble Tea program
- Card width = (terminalWidth - gaps) / numCards
- Frame offset: 4 chars (2 for borders + 2 for padding)
- Critical: cardStyle must account for frame when setting width

### Height Calculations
- Header: 1 line
- Stats row: max(cardHeights) + row gap (1 line)
- Log panel: remaining = terminalHeight - header - statsRow - gaps

---

## Testing Strategy

### Visual Testing
- Terminal widths: 80, 120, 160, 200+ columns
- Terminal heights: 24, 40, 60 lines
- Verify card stacking threshold
- Check border rendering at all widths

### Data Testing
- Empty states (no files discovered)
- Large counts (10k+ files)
- Long file paths in logs
- Rapid event streams
- Server connection loss

### Edge Cases
- Terminal resize during upload
- Very narrow terminal (< 80 cols)
- Very short terminal (< 20 lines)
- Unicode in filenames/logs

---

## References

**Source Files:**
- Layout: [internal/ui/platform/terminal/upload_dashboard.go](internal/ui/platform/terminal/upload_dashboard.go)
- Events: [internal/ui/messages/messages.go](internal/ui/messages/messages.go)
- Publishers: [app/upload/ui_pipeline.go](app/upload/ui_pipeline.go)

**Screen Capture:** Based on terminal output showing full dashboard with all sections populated during active upload operation.
