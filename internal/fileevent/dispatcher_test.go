package fileevent

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

// MockSink is a test sink that captures events for verification
type MockSink struct {
	mu     sync.Mutex
	events []MockEvent
}

type MockEvent struct {
	Code Code
	File string
	Size int64
	Args map[string]any
}

func (m *MockSink) HandleEvent(ctx context.Context, code Code, file string, size int64, args map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, MockEvent{
		Code: code,
		File: file,
		Size: size,
		Args: args,
	})
}

func (m *MockSink) GetEvents() []MockEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockEvent{}, m.events...)
}

func (m *MockSink) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}

func TestDispatcherRegistration(t *testing.T) {
	dispatcher := NewDispatcher()
	mock1 := &MockSink{}
	mock2 := &MockSink{}

	// Register sinks
	dispatcher.RegisterSink(mock1)
	dispatcher.RegisterSink(mock2)

	// Dispatch event
	ctx := context.Background()
	dispatcher.Dispatch(ctx, DiscoveredImage, "test.jpg", 1024, "album", "vacation")

	// Both sinks should receive the event
	events1 := mock1.GetEvents()
	events2 := mock2.GetEvents()

	if len(events1) != 1 {
		t.Errorf("Expected 1 event in mock1, got %d", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("Expected 1 event in mock2, got %d", len(events2))
	}

	// Verify event contents
	if events1[0].Code != DiscoveredImage {
		t.Errorf("Expected DiscoveredImage, got %v", events1[0].Code)
	}
	if events1[0].File != "test.jpg" {
		t.Errorf("Expected test.jpg, got %s", events1[0].File)
	}
	if events1[0].Size != 1024 {
		t.Errorf("Expected size 1024, got %d", events1[0].Size)
	}
	if events1[0].Args["album"] != "vacation" {
		t.Errorf("Expected album=vacation, got %v", events1[0].Args["album"])
	}
}

func TestDispatcherUnregistration(t *testing.T) {
	dispatcher := NewDispatcher()
	mock1 := &MockSink{}
	mock2 := &MockSink{}

	dispatcher.RegisterSink(mock1)
	dispatcher.RegisterSink(mock2)

	// Dispatch first event
	ctx := context.Background()
	dispatcher.Dispatch(ctx, DiscoveredImage, "test1.jpg", 1024)

	// Unregister mock1
	dispatcher.UnregisterSink(mock1)

	// Dispatch second event
	dispatcher.Dispatch(ctx, DiscoveredVideo, "test2.mp4", 2048)

	// mock1 should only have first event
	events1 := mock1.GetEvents()
	if len(events1) != 1 {
		t.Errorf("Expected 1 event in mock1, got %d", len(events1))
	}

	// mock2 should have both events
	events2 := mock2.GetEvents()
	if len(events2) != 2 {
		t.Errorf("Expected 2 events in mock2, got %d", len(events2))
	}
}

func TestDispatcherNilSink(t *testing.T) {
	dispatcher := NewDispatcher()

	// Should not panic when registering nil
	dispatcher.RegisterSink(nil)
	dispatcher.UnregisterSink(nil)

	// Should not panic when dispatching with no sinks
	ctx := context.Background()
	dispatcher.Dispatch(ctx, DiscoveredImage, "test.jpg", 1024)
}

func TestDispatcherArgsConversion(t *testing.T) {
	dispatcher := NewDispatcher()
	mock := &MockSink{}
	dispatcher.RegisterSink(mock)

	ctx := context.Background()

	// Test various arg types
	dispatcher.Dispatch(ctx, DiscoveredImage, "test.jpg", 1024,
		"string", "value",
		"int", 42,
		"bool", true,
		"error", "something went wrong",
	)

	events := mock.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	args := events[0].Args
	if args["string"] != "value" {
		t.Errorf("Expected string=value, got %v", args["string"])
	}
	if args["int"] != 42 {
		t.Errorf("Expected int=42, got %v", args["int"])
	}
	if args["bool"] != true {
		t.Errorf("Expected bool=true, got %v", args["bool"])
	}
	if args["error"] != "something went wrong" {
		t.Errorf("Expected error message, got %v", args["error"])
	}
}

func TestDispatcherOddArgs(t *testing.T) {
	dispatcher := NewDispatcher()
	mock := &MockSink{}
	dispatcher.RegisterSink(mock)

	ctx := context.Background()

	// Odd number of args (last one should be ignored)
	dispatcher.Dispatch(ctx, DiscoveredImage, "test.jpg", 1024,
		"key1", "value1",
		"key2", // No value
	)

	events := mock.GetEvents()
	args := events[0].Args

	if args["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", args["key1"])
	}
	if _, exists := args["key2"]; exists {
		t.Errorf("key2 should not exist in args")
	}
}

func TestDispatcherConcurrency(t *testing.T) {
	dispatcher := NewDispatcher()
	mock := &MockSink{}
	dispatcher.RegisterSink(mock)

	ctx := context.Background()
	const goroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Dispatch events concurrently
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				dispatcher.Dispatch(ctx, DiscoveredImage, "test.jpg", 1024, "goroutine", id, "event", j)
			}
		}(i)
	}

	wg.Wait()

	events := mock.GetEvents()
	expectedCount := goroutines * eventsPerGoroutine
	if len(events) != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, len(events))
	}
}

func TestSlogSink(t *testing.T) {
	// Create a buffer to capture log output
	var buf mockLogBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	sink := NewSlogSink(logger)
	ctx := context.Background()

	// Test normal info event
	sink.HandleEvent(ctx, DiscoveredImage, "test.jpg", 1024, map[string]any{"album": "vacation"})

	output := buf.String()
	if !contains(output, "discovered image") {
		t.Errorf("Expected log to contain 'discovered image', got: %s", output)
	}
	if !contains(output, "test.jpg") {
		t.Errorf("Expected log to contain 'test.jpg', got: %s", output)
	}
	if !contains(output, "album=vacation") {
		t.Errorf("Expected log to contain 'album=vacation', got: %s", output)
	}
}

func TestSlogSinkErrorLevel(t *testing.T) {
	var buf mockLogBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	sink := NewSlogSink(logger)
	ctx := context.Background()

	// Event with error should use ERROR level
	sink.HandleEvent(ctx, DiscoveredImage, "test.jpg", 1024, map[string]any{
		"error": "file corrupted",
	})

	output := buf.String()
	if !contains(output, "ERROR") && !contains(output, "level=ERROR") {
		t.Errorf("Expected log to contain ERROR level, got: %s", output)
	}
	if !contains(output, "file corrupted") {
		t.Errorf("Expected log to contain error message, got: %s", output)
	}
}

func TestSlogSinkNilLogger(t *testing.T) {
	sink := NewSlogSink(nil)
	ctx := context.Background()

	// Should not panic with nil logger
	sink.HandleEvent(ctx, DiscoveredImage, "test.jpg", 1024, nil)
}

func TestRecorderWithMultipleSinks(t *testing.T) {
	// Create recorder with slog
	var buf mockLogBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	recorder := New(logger)

	// Add mock sink
	mock := &MockSink{}
	recorder.RegisterSink(mock)

	ctx := context.Background()

	// Record event
	recorder.RecordWithSize(ctx, DiscoveredImage, mockFile("test.jpg"), 1024, "album", "vacation")

	// Mock sink should receive structured event
	events := mock.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event in mock, got %d", len(events))
	}
	if events[0].Code != DiscoveredImage {
		t.Errorf("Expected DiscoveredImage, got %v", events[0].Code)
	}
	if events[0].File != "test.jpg" {
		t.Errorf("Expected test.jpg, got %s", events[0].File)
	}
	if events[0].Size != 1024 {
		t.Errorf("Expected size 1024, got %d", events[0].Size)
	}

	// Slog should also have logged
	output := buf.String()
	if !contains(output, "discovered image") {
		t.Errorf("Expected log output to contain 'discovered image', got: %s", output)
	}
}

func TestRecorderSetLoggerUpdatesSink(t *testing.T) {
	var buf1 mockLogBuffer
	logger1 := slog.New(slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo}))
	recorder := New(logger1)

	ctx := context.Background()

	// Record with first logger
	recorder.RecordWithSize(ctx, DiscoveredImage, mockFile("test1.jpg"), 1024)

	// Change logger
	var buf2 mockLogBuffer
	logger2 := slog.New(slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo}))
	recorder.SetLogger(logger2)

	// Record with second logger
	recorder.RecordWithSize(ctx, DiscoveredVideo, mockFile("test2.mp4"), 2048)

	// First buffer should only have first event
	output1 := buf1.String()
	if !contains(output1, "test1.jpg") {
		t.Errorf("Expected first buffer to contain test1.jpg")
	}
	if contains(output1, "test2.mp4") {
		t.Errorf("Expected first buffer to NOT contain test2.mp4")
	}

	// Second buffer should only have second event
	output2 := buf2.String()
	if contains(output2, "test1.jpg") {
		t.Errorf("Expected second buffer to NOT contain test1.jpg")
	}
	if !contains(output2, "test2.mp4") {
		t.Errorf("Expected second buffer to contain test2.mp4")
	}
}

// Helper types and functions

type mockLogBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *mockLogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *mockLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

type mockFile string

func (f mockFile) LogValue() slog.Value {
	return slog.StringValue(string(f))
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
