package script

import (
	"log"
	"time"
)

// Event types based on the original C++ engine
const (
	EventBG = iota
	EventMovie
	EventSE
	EventBlackFade
	EventWhiteFade
	EventText
	EventBGM
	EventVoice
	EventNone
)

// Event states
const (
	EventWait = iota
	EventRun
	EventEnd
)

// Event represents a script event
type Event struct {
	Type       int
	State      int
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Direction  bool // true for IN, false for OUT
	FloatValue float64
	Data       []string
	NextState  bool
}

// Engine handles script execution and event processing
type Engine struct {
	events    []*Event
	running   bool
	startTime time.Time
}

// NewEngine creates a new script engine
func NewEngine() *Engine {
	return &Engine{
		events:  make([]*Event, 0),
		running: false,
	}
}

// Init initializes the script engine
func (e *Engine) Init() error {
	log.Println("Script engine initialized")
	return nil
}

// Start starts the script engine
func (e *Engine) Start() {
	e.running = true
	e.startTime = time.Now()
	log.Println("Script engine started")
}

// Stop stops the script engine
func (e *Engine) Stop() {
	e.running = false
	log.Println("Script engine stopped")
}

// AddEvent adds a new event to the script engine
func (e *Engine) AddEvent(eventType int, data ...string) {
	event := &Event{
		Type:      eventType,
		State:     EventWait,
		Data:      data,
		Direction: true,
	}

	// Parse timing information if provided
	if len(data) >= 3 {
		// data[1] should be start time, data[2] should be duration
		// For now, use simple duration parsing
		event.Duration = time.Second * 2                         // Default 2 seconds
		event.StartTime = time.Now().Add(time.Millisecond * 100) // Start in 100ms
		event.EndTime = event.StartTime.Add(event.Duration)

		// Parse direction from data[0]
		if len(data) > 0 {
			event.Direction = (data[0] == "IN")
		}
	}

	e.events = append(e.events, event)
	log.Printf("Added event type %d with %d data elements", eventType, len(data))
}

// Update processes script events
func (e *Engine) Update() error {
	if !e.running {
		return nil
	}

	currentTime := time.Now()

	for _, event := range e.events {
		if event.State == EventEnd {
			continue
		}

		// Check if event should start
		if event.State == EventWait && currentTime.After(event.StartTime) {
			event.State = EventRun
			e.startEvent(event)
		}

		// Process running events
		if event.State == EventRun {
			if currentTime.After(event.EndTime) {
				// Event finished
				event.State = EventEnd
				e.endEvent(event)
			} else {
				// Event in progress
				e.updateEvent(event, currentTime)
			}
		}
	}

	return nil
}

// startEvent handles event start
func (e *Engine) startEvent(event *Event) {
	switch event.Type {
	case EventBlackFade, EventWhiteFade:
		if event.Direction {
			event.FloatValue = 0.0 // Fade in starts transparent
		} else {
			event.FloatValue = 1.0 // Fade out starts opaque
		}
		log.Printf("Started fade event, direction: %v", event.Direction)

	case EventBG:
		log.Printf("Started background event with data: %v", event.Data)

	case EventBGM:
		log.Printf("Started BGM event with data: %v", event.Data)

	case EventSE:
		log.Printf("Started sound effect event with data: %v", event.Data)

	case EventText:
		log.Printf("Started text event with data: %v", event.Data)
	}
}

// updateEvent handles event progress
func (e *Engine) updateEvent(event *Event, currentTime time.Time) {
	elapsed := currentTime.Sub(event.StartTime)
	progress := float64(elapsed) / float64(event.Duration)

	if progress > 1.0 {
		progress = 1.0
	}

	switch event.Type {
	case EventBlackFade, EventWhiteFade:
		if event.Direction {
			// Fade in: go from 1.0 to 0.0
			event.FloatValue = 1.0 - progress
		} else {
			// Fade out: go from 0.0 to 1.0
			event.FloatValue = progress
		}
	}
}

// endEvent handles event completion
func (e *Engine) endEvent(event *Event) {
	switch event.Type {
	case EventBlackFade, EventWhiteFade:
		if event.Direction {
			event.FloatValue = 0.0 // Fade in complete (transparent)
		} else {
			event.FloatValue = 1.0 // Fade out complete (opaque)
		}
		log.Printf("Completed fade event, final value: %f", event.FloatValue)

	case EventBG:
		log.Println("Completed background event")

	case EventBGM:
		log.Println("Completed BGM event")
	}

	// Check if this event should trigger a state change
	if event.NextState {
		log.Println("Event triggered next state")
	}
}

// Clear removes all events
func (e *Engine) Clear() {
	e.events = e.events[:0]
	log.Println("Cleared all script events")
}

// GetEvents returns the current events (for external systems to read)
func (e *Engine) GetEvents() []*Event {
	return e.events
}

// IsRunning returns whether the engine is running
func (e *Engine) IsRunning() bool {
	return e.running
}
