package golly

import (
	"fmt"
	"testing"

	"github.com/golly-go/golly/errors"
	"github.com/golly-go/golly/utils"
	"github.com/stretchr/testify/assert"
)

var (
	TestError = errors.Error{Key: "TEST.ERROR"}.NewError(fmt.Errorf("%s", "test"))
)

func TestEventDispatch(t *testing.T) {

	t.Run("fires successfully", func(t *testing.T) {

		var cnt = 0
		root := EventChain{}

		root.
			Namespace("test").
			Add("event", func(ctx Context, e Event) error {
				cnt += 1
				return nil
			}).
			Add("event", func(ctx Context, e Event) error {
				cnt += 1
				return nil
			})

		assert.Len(t, root.children, 1, "expected single child")

		t.Run("it should dispatch all handlers", func(t *testing.T) {
			err := root.Dispatch(Context{}, "test:event", struct{}{})

			assert.NoError(t, err)
			assert.Equal(t, 2, cnt)
		})
	})

	t.Run("fires unsuccesfully in the middle", func(t *testing.T) {

		var cnt = 0
		root := EventChain{}

		root.
			Add("test:event", func(ctx Context, e Event) error {
				cnt += 1
				return nil
			}).
			Add("test:event", func(ctx Context, e Event) error {
				cnt += 1
				return TestError
			}).
			Add("test:event", func(ctx Context, e Event) error {
				cnt += 1
				return nil
			})
		assert.Len(t, root.children, 1, "expected single child")

		t.Run("it should dispatch all handlers", func(t *testing.T) {
			err := root.Dispatch(Context{}, "test:event", struct{}{})

			assert.Error(t, err)
			assert.Equal(t, 2, cnt)
		})
	})

}

// TestRemoveHandlerTableDriven tests the RemoveHandler function in a table-driven manner
func TestRemoveHandlerTableDriven(t *testing.T) {
	// Define a struct for test cases
	type testCase struct {
		name          string
		setupHandlers []EventHandlerFunc
		removeHandler EventHandlerFunc
		expectedLen   int
	}

	// Define some handler functions for testing
	handler1 := func(Context, Event) error { return nil }
	handler2 := func(Context, Event) error { return nil }
	handler3 := func(Context, Event) error { return nil }

	// Create test cases
	testCases := []testCase{
		{
			name:          "Remove handler1 from two handlers",
			setupHandlers: []EventHandlerFunc{handler1, handler2},
			removeHandler: handler1,
			expectedLen:   1,
		},
		{
			name:          "Remove handler2 from three handlers",
			setupHandlers: []EventHandlerFunc{handler1, handler2, handler3},
			removeHandler: handler2,
			expectedLen:   2,
		},
		{
			name:          "Remove non-existent handler from two handlers",
			setupHandlers: []EventHandlerFunc{handler1, handler2},
			removeHandler: handler3,
			expectedLen:   2,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an EventChain and set up handlers
			eventChain := &EventChain{}
			eventChain.handlers = tc.setupHandlers

			// Remove the specified handler
			eventChain.remove(tc.removeHandler)

			// Assert the results
			assert.Len(t, eventChain.handlers, tc.expectedLen, tc.name)
		})
	}
}

func TestEventChain_Resolve(t *testing.T) {
	// Define a struct for test cases
	type testCase struct {
		name         string
		initialChain func(*EventChain) *EventChain
		path         string
		expectedName string
	}

	// Define test cases
	testCases := []testCase{
		{
			name: "Resolve single level",
			initialChain: func(evl *EventChain) *EventChain {
				return evl.On("child1", NoOpEventHandler)
			},
			path:         "child1",
			expectedName: "child1",
		},
		{
			name: "Resolve multiple levels",
			initialChain: func(evl *EventChain) *EventChain {
				return evl.On("grandchild1:child1", NoOpEventHandler)
			},
			path:         "grandchild1:child1",
			expectedName: "child1",
		},
		{
			name: "Resolve with namespace levels",
			initialChain: func(evl *EventChain) *EventChain {
				return evl.Namespace("grandchild1").On("child1", NoOpEventHandler)
			},
			path:         "grandchild1:child1",
			expectedName: "child1",
		},
		{
			name: "Resolve multiple levels",
			initialChain: func(evl *EventChain) *EventChain {
				return evl.On("grandchild1:child1", NoOpEventHandler)
			},
			path:         "grandchild1",
			expectedName: "grandchild1",
		},
		{
			name:         "Resolve empty path",
			path:         "",
			expectedName: "",
		},
		// More test cases can be added here
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evc := &EventChain{}
			if tc.initialChain != nil {
				evc = tc.initialChain(evc)
			}

			// Resolve the path
			resolvedChain := evc.resolve(tc.path)

			// Assert that the resolved chain has the correct name
			assert.Equal(t, utils.WildcardString(tc.expectedName), resolvedChain.Name, tc.name)
		})
	}
}
