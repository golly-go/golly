package golly

import (
	"context"
	"reflect"
	"testing"
)

func TestEventManagerUnregister(t *testing.T) {
	handler1 := func(ctx context.Context, data any) {}
	handler2 := func(ctx context.Context, data any) {}
	handler3 := func(ctx context.Context, data any) {}

	tests := []struct {
		name           string
		setup          func(*EventManager)             // Setup the event manager state
		unregisterName string                          // Event name to unregister from
		unregisterFunc EventFunc                       // Function to unregister
		validate       func(*testing.T, *EventManager) // Validation function
	}{
		{
			name: "Unregister single handler from event",
			setup: func(em *EventManager) {
				em.Register("test.event", handler1)
			},
			unregisterName: "test.event",
			unregisterFunc: handler1,
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 0 {
					t.Errorf("Expected 0 handlers after unregister, got %d", len(handlers))
				}
			},
		},
		{
			name: "Unregister one handler when multiple exist",
			setup: func(em *EventManager) {
				em.Register("test.event", handler1)
				em.Register("test.event", handler2)
				em.Register("test.event", handler3)
			},
			unregisterName: "test.event",
			unregisterFunc: handler2,
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 2 {
					t.Errorf("Expected 2 handlers after unregister, got %d", len(handlers))
				}
				// Verify handler2 is not in the list
				handler2Ptr := reflect.ValueOf(handler2).Pointer()
				for _, h := range handlers {
					if reflect.ValueOf(h).Pointer() == handler2Ptr {
						t.Error("handler2 should have been removed but is still present")
					}
				}
			},
		},
		{
			name: "Unregister non-existent handler from existing event",
			setup: func(em *EventManager) {
				em.Register("test.event", handler1)
			},
			unregisterName: "test.event",
			unregisterFunc: handler2, // This was never registered
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 1 {
					t.Errorf("Expected 1 handler after failed unregister, got %d", len(handlers))
				}
			},
		},
		{
			name: "Unregister from non-existent event",
			setup: func(em *EventManager) {
				em.Register("other.event", handler1)
			},
			unregisterName: "nonexistent.event",
			unregisterFunc: handler1,
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["other.event"]
				nonExistentHandlers := em.events["nonexistent.event"]
				em.mu.RUnlock()
				if len(handlers) != 1 {
					t.Errorf("Expected original event to remain unchanged, got %d handlers", len(handlers))
				}
				if len(nonExistentHandlers) != 0 {
					t.Errorf("Expected non-existent event to have 0 handlers, got %d", len(nonExistentHandlers))
				}
			},
		},
		{
			name: "Unregister same handler twice",
			setup: func(em *EventManager) {
				em.Register("test.event", handler1)
				em.Register("test.event", handler2)
				// First unregister
				em.Unregister("test.event", handler1)
			},
			unregisterName: "test.event",
			unregisterFunc: handler1, // Try to unregister again
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 1 {
					t.Errorf("Expected 1 handler after double unregister, got %d", len(handlers))
				}
			},
		},
		{
			name: "Unregister all handlers one by one",
			setup: func(em *EventManager) {
				em.Register("test.event", handler1)
				em.Register("test.event", handler2)
				em.Register("test.event", handler3)
			},
			unregisterName: "test.event",
			unregisterFunc: handler1,
			validate: func(t *testing.T, em *EventManager) {
				// Continue unregistering the remaining handlers
				em.Unregister("test.event", handler2)
				em.Unregister("test.event", handler3)

				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 0 {
					t.Errorf("Expected 0 handlers after unregistering all, got %d", len(handlers))
				}
			},
		},
		{
			name: "Unregister from empty event manager",
			setup: func(em *EventManager) {
				// No setup - empty event manager
			},
			unregisterName: "test.event",
			unregisterFunc: handler1,
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers := em.events["test.event"]
				em.mu.RUnlock()
				if len(handlers) != 0 {
					t.Errorf("Expected 0 handlers in empty event manager, got %d", len(handlers))
				}
			},
		},
		{
			name: "Unregister preserves other events",
			setup: func(em *EventManager) {
				em.Register("event.one", handler1)
				em.Register("event.two", handler2)
				em.Register("event.three", handler3)
			},
			unregisterName: "event.two",
			unregisterFunc: handler2,
			validate: func(t *testing.T, em *EventManager) {
				em.mu.RLock()
				handlers1 := em.events["event.one"]
				handlers2 := em.events["event.two"]
				handlers3 := em.events["event.three"]
				em.mu.RUnlock()

				if len(handlers1) != 1 {
					t.Errorf("Expected event.one to have 1 handler, got %d", len(handlers1))
				}
				if len(handlers2) != 0 {
					t.Errorf("Expected event.two to have 0 handlers, got %d", len(handlers2))
				}
				if len(handlers3) != 1 {
					t.Errorf("Expected event.three to have 1 handler, got %d", len(handlers3))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh event manager for each test
			em := NewEventManager()

			// Setup the test state
			if tt.setup != nil {
				tt.setup(em)
			}

			// Perform the unregister operation
			result := em.Unregister(tt.unregisterName, tt.unregisterFunc)

			// Verify the method returns the event manager (fluent interface)
			if result != em {
				t.Error("Unregister should return the EventManager instance for method chaining")
			}

			// Run custom validation
			if tt.validate != nil {
				tt.validate(t, em)
			}
		})
	}
}

// Test event type for dispatch testing
type TestDispatchEvent struct {
	Name string
}

// Test that we can dispatch events properly after unregistering
func TestEventManagerUnregisterWithDispatch(t *testing.T) {
	em := NewEventManager()
	gctx := NewContext(context.Background())

	callCount := 0
	handler1 := func(ctx context.Context, data any) { callCount++ }
	handler2 := func(ctx context.Context, data any) { callCount += 10 }

	// Register both handlers - use the type name that Dispatch will generate
	eventName := TypeNoPtr(TestDispatchEvent{}).String()
	em.Register(eventName, handler1)
	em.Register(eventName, handler2)

	// Dispatch - should call both handlers
	em.Dispatch(gctx, TestDispatchEvent{Name: "test"})
	if callCount != 11 { // 1 + 10
		t.Errorf("Expected callCount to be 11 after first dispatch, got %d", callCount)
	}

	// Unregister one handler
	em.Unregister(eventName, handler1)

	// Reset and dispatch again - should only call handler2
	callCount = 0
	em.Dispatch(gctx, TestDispatchEvent{Name: "test"})
	if callCount != 10 { // Only handler2 should be called
		t.Errorf("Expected callCount to be 10 after unregister and dispatch, got %d", callCount)
	}
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

// BenchmarkEventManagerDispatch tests the performance of the Dispatch method.
func BenchmarkEventManagerDispatch(b *testing.B) {
	em := &EventManager{}

	// Sample event type
	type SampleEvent struct {
		ID int
	}

	// Simple handler function that does minimal work
	handler := func(ctx context.Context, data any) {}

	// Register handlers for the event
	em.Register("golly.SampleEvent", handler)
	em.Register("golly.SampleEvent", handler)
	em.Register("golly.SampleEvent", handler)

	// Create a context and sample event
	gctx := NewContext(context.Background())
	event := SampleEvent{ID: 1}

	b.Run("Dispatch", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			em.Dispatch(gctx, event)
		}
	})
}

// Benchmark with multiple event types to simulate diverse workloads
func BenchmarkEventManagerDispatch_MultiEvent(b *testing.B) {
	em := &EventManager{}

	type EventA struct{}
	type EventB struct{}
	type EventC struct{}

	handler := func(ctx context.Context, data any) {}

	// Register different handlers
	em.Register("golly.EventA", handler)
	em.Register("golly.EventB", handler)
	em.Register("golly.EventC", handler)

	// Create a context and sample events
	gctx := NewContext(context.Background())
	eventA := EventA{}
	eventB := EventB{}
	eventC := EventC{}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("Dispatch EventA", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			em.Dispatch(gctx, eventA)
		}
	})

	b.Run("Dispatch EventB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			em.Dispatch(gctx, eventB)
		}
	})

	b.Run("Dispatch EventC", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			em.Dispatch(gctx, eventC)
		}
	})
}
