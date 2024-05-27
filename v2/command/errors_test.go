package command

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestError(t *testing.T) {
	var err error
	err = &GoxHttpError{}
	assert.Error(t, err)

	if e, ok := err.(*GoxHttpError); ok {
		assert.False(t, e.Is5xx())
	} else {
		assert.Fail(t, "expected to be GoxHttpError")
	}
}
