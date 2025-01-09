package golly

import (
	"context"
	"testing"
)

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
	handler := func(_ *Context, _ *Event) {}

	// Register handlers for the event
	em.Register("golly.SampleEvent", handler)
	em.Register("golly.SampleEvent", handler)
	em.Register("golly.SampleEvent", handler)

	// Create a context and sample event
	gctx := NewContext(context.Background())
	event := SampleEvent{ID: 1}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		em.Dispatch(gctx, event)
	}
}

// Benchmark with multiple event types to simulate diverse workloads
func BenchmarkEventManagerDispatch_MultiEvent(b *testing.B) {
	em := &EventManager{}

	type EventA struct{}
	type EventB struct{}
	type EventC struct{}

	handler := func(_ *Context, _ *Event) {}

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
