# Script Engine Analysis - C++ vs Go Implementation

## Critical Differences Found

### 1. Event Data Structure
**C++ (Original):**
```cpp
typedef struct {
    EEventType  type;
    char  data[10][100];        // Fixed array of 10 strings, 100 chars each
    char  start[10];            // "MM:SS:FF" format
    char  end[10];              // "MM:SS:FF" format
    uint32_t    t_start;        // Absolute timestamp in ms
    uint32_t    t_end;          // Absolute timestamp in ms
    uint32_t    t_delta;        // Duration = t_end - t_start
    bool        dir;            // Direction: true=IN, false=OUT
    float       f_val;          // Current fade value
    void        *mem;           // Memory pointer for audio/video
    bool        next_state;     // Trigger state change on completion
    EEventState state;          // WAIT, RUN, END
} event_t;
```

**Go (Current):**
```go
type Event struct {
    Type       int
    State      int
    StartTime  time.Time       // Different timing approach
    EndTime    time.Time
    Duration   time.Duration   // Relative duration
    Direction  bool
    FloatValue float64
    Data       []string        // Dynamic slice vs fixed array
    NextState  bool
}
```

### 2. Time Format Parsing
**C++ (Original):**
- Uses "MM:SS:FF" format (Minutes:Seconds:Frames at 60fps)
- `str2time()` converts to absolute milliseconds
- Frames: `usec*16.6667` (60fps = 16.67ms per frame)

**Go (Current):**
- Uses Go's `time.Duration` 
- No support for frame-based timing

### 3. Event Field Indices (C++ Constants)
```cpp
#define EVENT_FIED_FILE 0        // File name
#define EVENT_FIED_LAYER_IDX 1   // Layer index
#define EVENT_FIED_TITLE 2       // Title text
#define EVENT_FIED_TEXT 3        // Dialog text  
#define EVENT_FIED_PERS 4        // Person/character
#define EVENT_EX_START 5         // Extended start
```

### 4. Fade Calculations
**C++ (Original):**
```cpp
// Fade IN: progress from 0.0 to 1.0
if (event->dir)
    event->f_val = (this->m_time - event->t_start * 1.) / event->t_delta;
// Fade OUT: progress from 1.0 to 0.0  
else
    event->f_val = (event->t_end - this->m_time * 1.) / event->t_delta;
```

**Go (Current):**
```go
// Simplified linear interpolation
if event.Direction {
    event.FloatValue = 1.0 - progress  // Fade in: 1.0 to 0.0
} else {
    event.FloatValue = progress        // Fade out: 0.0 to 1.0
}
```

### 5. Script File Parsing
**C++ (Original):**
- Parses script files using `Parser` class
- Tab-separated values in specific format
- Supports event types: CreateBG, PlayMovie, PlaySe, PrintText, BlackFade, WhiteFade, PlayBgm, PlayVoice

**Go (Current):**
- No script file parsing implemented
- Manual event creation only

## Required Fixes

1. **Fix Event structure** to match C++ field layout
2. **Implement MM:SS:FF time parsing** 
3. **Add script file parsing** with tab-separated format
4. **Fix fade calculations** to match C++ formulas
5. **Add proper event field constants**
6. **Implement memory management** for audio/video resources
7. **Add state transition triggering** via menu system
