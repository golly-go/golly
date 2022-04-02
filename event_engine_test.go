package golly

import (
	"fmt"
	"testing"

	"github.com/slimloans/golly/errors"
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
			Add("event", func(e Event) error {
				cnt += 1
				return nil
			}).
			Add("event", func(e Event) error {
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
			Add("test:event", func(e Event) error {
				cnt += 1
				return nil
			}).
			Add("test:event", func(e Event) error {
				cnt += 1
				return TestError
			}).
			Add("test:event", func(e Event) error {
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
