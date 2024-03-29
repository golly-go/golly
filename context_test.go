package golly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextData(t *testing.T) {
	examples := []string{"1", "2", "3", "4", "5", "!@3412341234123"}

	c := NewContext(context.TODO())

	for _, example := range examples {
		c.Set(example, example)

		s, ok := c.Get(example)

		assert.True(t, ok)
		assert.Equal(t, example, s.(string))
	}

}
