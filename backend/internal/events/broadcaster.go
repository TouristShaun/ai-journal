package events

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventEntryProcessing EventType = "entry.processing"
	EventEntryProcessed  EventType = "entry.processed"
	EventEntryFailed     EventType = "entry.failed"
)

// Event represents a server-sent event
type Event struct {
	Type      EventType      `json:"type"`
	EntryID   string         `json:"entry_id"`
	Data      interface{}    `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// Client represents a connected SSE client
type Client struct {
	ID     string
	Events chan *Event
	Done   chan bool
}

// Broadcaster manages SSE event broadcasting
type Broadcaster struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Event
	mu         sync.RWMutex
}

// NewBroadcaster creates a new event broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Event),
	}
}

// Start begins the broadcaster event loop
func (b *Broadcaster) Start() {
	go func() {
		for {
			select {
			case client := <-b.register:
				b.mu.Lock()
				b.clients[client.ID] = client
				b.mu.Unlock()
				log.Printf("SSE client registered: %s", client.ID)

			case client := <-b.unregister:
				b.mu.Lock()
				if _, ok := b.clients[client.ID]; ok {
					close(client.Events)
					delete(b.clients, client.ID)
				}
				b.mu.Unlock()
				log.Printf("SSE client unregistered: %s", client.ID)

			case event := <-b.broadcast:
				b.mu.RLock()
				for _, client := range b.clients {
					select {
					case client.Events <- event:
						// Event sent successfully
					default:
						// Client is slow, skip this event
						log.Printf("Skipping event for slow client: %s", client.ID)
					}
				}
				b.mu.RUnlock()
			}
		}
	}()
}

// RegisterClient registers a new SSE client
func (b *Broadcaster) RegisterClient(clientID string) *Client {
	client := &Client{
		ID:     clientID,
		Events: make(chan *Event, 10), // Buffer to handle bursts
		Done:   make(chan bool),
	}
	b.register <- client
	return client
}

// UnregisterClient removes a client from the broadcaster
func (b *Broadcaster) UnregisterClient(client *Client) {
	b.unregister <- client
}

// SendEvent broadcasts an event to all connected clients
func (b *Broadcaster) SendEvent(eventType EventType, entryID string, data interface{}) {
	event := &Event{
		Type:      eventType,
		EntryID:   entryID,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	select {
	case b.broadcast <- event:
		// Event queued for broadcast
	default:
		log.Printf("Event broadcast channel full, dropping event")
	}
}

// FormatSSE formats an event for SSE transmission
func FormatSSE(event *Event) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("data: %s\n\n", string(data)), nil
}