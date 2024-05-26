package goxHttpApi

import (
	"context"
	"errors"
	"github.com/devlibx/gox-base/test"
	"github.com/devlibx/gox-http/command"
	"github.com/devlibx/gox-http/testhelper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoxHttpContext_WithNonExistingApiName(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testhelper.GetTestConfig(&config)
	assert.NoError(t, err)

	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	_, err = goxHttpCtx.Execute(context.TODO(), "badName", nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotRegisteredForApi))
}
